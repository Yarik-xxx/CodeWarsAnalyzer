package main

import (
	"CodeWarsAnalyzer/cwapi"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

const chunk = 5

var numNewElements = 0

func main() {
	var cache cwapi.Cache

	// Инициализация кэша
	log.Printf("Открытие кэша...\n")
	err := cache.Init()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Кэш открыт")

	groupID := make([]string, 0, chunk) // слайс из 10 id
	for idKata, _ := range cache.CacheData {
		// Сбор ID до нужного количества
		if len(groupID) < chunk {
			groupID = append(groupID, idKata)
			continue
		}

		// Отправка запросов всей группы ID
		log.Printf("Отправка запросов группы\n")
		var wg sync.WaitGroup
		var mt sync.Mutex

		for _, idKata = range groupID {
			wg.Add(1)

			go func(idKata string, cache *cwapi.Cache, wg *sync.WaitGroup) {
				defer wg.Done()

				if err := updateKataInfo(idKata, cache, &mt); err != nil {
					log.Printf("An error occurred while receiving kyu kata: %s\n", err)
				}
			}(idKata, &cache, &wg)
		}
		wg.Wait()
	}
	// Очистка слайса
	groupID = make([]string, 0, chunk)

	// Обновление файла
	if numNewElements >= 10 {
		log.Printf("Обновление файла\n")
		err := cache.UpdateFileCache()
		if err != nil {
			log.Fatal(err)
		}
		numNewElements = 0
	}
}

func updateKataInfo(id string, cache *cwapi.Cache, mt *sync.Mutex) error {
	// Отправка запроса
	body, err := getRequest(fmt.Sprintf("https://www.codewars.com/api/v1/code-challenges/%s", id))
	if err != nil {
		return err
	}

	// Десерелизация данных
	var kyuR cwapi.KataKyuStruct
	if err = json.Unmarshal(body, &kyuR); err != nil {
		return err
	}

	// Проверка пустого значения
	if kyuR.Rank.Name == "" {
		kyuR.Rank.Name = "Beta"
	}

	// Обновления кэша в оперативной памяти
	if kyuR.Rank.Name != cache.CacheData[id] {
		log.Printf("Update %s. New kyu: %s\n", id, kyuR.Rank.Name)
		mt.Lock()
		cache.UpdateCache(id, kyuR.Rank.Name)
		numNewElements++
		mt.Unlock()
	}

	return nil

}

func getRequest(url string) ([]byte, error) {
	// Отправка запроса
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	// Повторная отправка в случае 429 ошибки
	attempt := time.Duration(2)
	for resp.StatusCode == 429 {
		log.Printf("Stop request (%d seconds). Status code 429 <url %s>", attempt, url)
		time.Sleep(attempt * time.Second)
		attempt = (attempt * 3) / 2
		resp, err = http.Get(url)
	}

	if resp.StatusCode != 200 {
		return nil, errors.New("status code not 200")
	}

	// Чтение полученных данных
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
