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

var is429 = false
var mt sync.Mutex

func main() {
	var mtCache sync.Mutex

	cache := cacheInit()

	countCache := len(cache.CacheData)
	countIter := 0
	countUpdate := 0
	fmt.Printf("Estimated update time: %.2f minutes\n", float32(countCache)/200)

	for id, oldKyu := range cache.CacheData {
		for is429 {
			time.Sleep(10 * time.Millisecond)
		}

		countIter++
		fmt.Printf(
			"\rReady for %.2f%% (updates: %d, left kata: %d)",
			float32(countIter)/float32(countCache)*100, countUpdate, countCache-countIter)

		// Время было выбрано из эксперементального расчета максимального колличества запросов в минуту (200/мин или 1/300 мсек)
		time.Sleep(300 * time.Millisecond)

		go func(id string, oldKyu string) {
			newKyu, ok := comparisonKyu(id, oldKyu)
			if ok {
				countUpdate++
				log.Printf("Update kyu (%s): %s --> %s", id, oldKyu, newKyu)
				mtCache.Lock()
				cache.UpdateCache(id, newKyu)
				mtCache.Unlock()
			}
		}(id, oldKyu)
	}
	err := cache.UpdateFileCache()
	if err != nil {
		log.Fatal(err)
	}

}

func comparisonKyu(id string, oldKyu string) (string, bool) {
	newKyu, err := getKyu(id)
	if err != nil {
		log.Printf("%s (error): %s", id, err)
		return oldKyu, false
	}

	if newKyu != oldKyu {
		return newKyu, true
	}

	return oldKyu, false
}

func getKyu(id string) (string, error) {
	body, err := getRequest(fmt.Sprintf("https://www.codewars.com/api/v1/code-challenges/%s", id))
	if err != nil {
		return "", err
	}

	// Десерелизация данных
	var kyuR cwapi.KataKyuStruct
	if err = json.Unmarshal(body, &kyuR); err != nil {
		return "", err
	}

	// Проверка пустого значения
	if kyuR.Rank.Name == "" {
		kyuR.Rank.Name = "Beta"
	}

	return kyuR.Rank.Name, nil
}

func getRequest(url string) ([]byte, error) {
	// Отправка запроса
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	// Повторная отправка в случае 429 ошибки
	attempt := time.Duration(10)
	for resp.StatusCode == 429 {
		mt.Lock()
		is429 = true
		mt.Unlock()

		time.Sleep(attempt * time.Second)
		resp, err = http.Get(url)
		if resp.StatusCode != 429 {
			mt.Lock()
			is429 = false
			mt.Unlock()
		}
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

func cacheInit() cwapi.Cache {
	var cache cwapi.Cache

	err := cache.Init()
	if err != nil {
		log.Fatal(err)
	}

	return cache
}
