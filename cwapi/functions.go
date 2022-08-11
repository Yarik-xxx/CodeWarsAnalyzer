package cwapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"
)

// GetAllInfoUser получает всю информацию о пользователе
func GetAllInfoUser(username string, cache *Cache) AllInfoUser {
	var response AllInfoUser

	// Получение основной информации
	err := response.getBaseInfoUser(username)
	if err != nil {
		response.checkError(err)
		return response
	}

	log.Printf("Request received <%s>\n", response.Username)

	// Получение списка выполненных кат по ЯП
	err = response.getCompletedKataUser(username)
	if err != nil {
		response.checkError(err)
		return response
	}

	// Подсчет выполненных кат по ЯП
	err = response.counterLangCompleted(cache)
	if err != nil {
		response.checkError(err)
		return response
	}

	return response
}

// getBaseInfoUser получает основную информацию о пользователе
func (s *AllInfoUser) getBaseInfoUser(username string) error {
	// Отправка запроса
	body, err := getRequest(fmt.Sprintf(baseInfo, username))
	if err != nil {
		return err
	}

	// Десериализация данных
	err = json.Unmarshal(body, s)
	if err != nil {
		return errGenResponse
	}
	return nil
}

// getCompletedKataUser получает список выполненных кат по ЯП
func (s *AllInfoUser) getCompletedKataUser(username string) error {
	// Определение количества страниц
	totalPages, err := completedKataTotalPages(username)
	if err != nil {
		return err
	}

	// Инициализация map
	s.LanguagesTotalCompleted = make(map[string][]struct {
		Id   string
		Name string
	})

	// Получение выполненных кат со всех страниц
	responseChanel := make(chan responseByte)
	go completedKataDataPages(username, totalPages, responseChanel)

	// Чтение из канала данных и обновление структуры
	for response := range responseChanel {
		if response.err != nil {
			return err
		}
		if err = s.updateLanguagesTotalCompleted(response.data); err != nil {
			return err
		}
	}

	return nil
}

// completedKataTotalPages получает количество страниц выполненных кат для username.
// Вызывается из getCompletedKataUser
func completedKataTotalPages(username string) (int, error) {
	body, err := getRequest(fmt.Sprintf(listCompletedKata, username, 0))
	if err != nil {
		return 0, err
	}
	var countPages struct{ TotalPages int }
	err = json.Unmarshal(body, &countPages)
	if err != nil {
		return 0, errGenResponse
	}

	return countPages.TotalPages, nil
}

// completedKataDataPages выполняет запросы по всем страницам выполненных кат и записывает их в канал
// Вызывается из getCompletedKataUser
func completedKataDataPages(username string, totalPages int, responseChanel chan responseByte) {
	// Запрос страниц порциями
	for start, stop := 0, chunk; start < totalPages; stop += chunk {
		if stop >= totalPages {
			stop = totalPages
		}

		var wg sync.WaitGroup
		for page := start; page < stop; page++ {
			wg.Add(1)

			// Получение данных одной страницы
			go func(username string, page int, c chan responseByte, wg *sync.WaitGroup) {
				defer wg.Done()
				body, err := getRequest(fmt.Sprintf(listCompletedKata, username, page))
				c <- responseByte{err: err, data: body}
			}(username, page, responseChanel, &wg)
		}
		wg.Wait()
		start = stop
	}

	close(responseChanel)
}

// updateLanguagesTotalCompleted обновляет и струтрирует данные выполненных кат
// Вызывается из getCompletedKataUser
func (s *AllInfoUser) updateLanguagesTotalCompleted(b []byte) error {
	var pageData CompletedKataStruct
	err := json.Unmarshal(b, &pageData)
	if err != nil {
		return errGenResponse
	}

	// Добавление выполненных кат по ЯП
	for _, kata := range pageData.Data {
		for _, lang := range kata.CompletedLanguages {
			s.LanguagesTotalCompleted[lang] = append(s.LanguagesTotalCompleted[lang], struct {
				Id   string
				Name string
			}{Id: kata.Id, Name: kata.Name})
		}
	}
	return nil
}

// counterLangCompleted подсчитывает количество выполненных кат по kyu
func (s *AllInfoUser) counterLangCompleted(cache *Cache) error {
	s.CountLangCompleted = make(map[string]map[string]int)

	pendingRequest := make([]counterStruct, 0) // срез для значений отсутсвующих в кэше

	// Обработка кэшированных данных
	for lang, data := range s.LanguagesTotalCompleted {
		s.CountLangCompleted[lang] = make(map[string]int)
		for _, kata := range data {
			// Проверка на наличие в кэше
			kyu, ok := cache.CacheData[kata.Id]
			if !ok {
				pendingRequest = append(pendingRequest, counterStruct{language: lang, kataId: kata.Id})
			}
			s.CountLangCompleted[lang][kyu]++
		}
	}

	// Обработка порциями не кешированных данных
	var wg sync.WaitGroup
	var mt1 sync.Mutex
	var mt2 sync.Mutex

	for start, stop, lenSlice := 0, chunk, len(pendingRequest); start < lenSlice; stop += chunk {
		if stop >= lenSlice {
			stop = lenSlice
		}

		for _, kata := range pendingRequest[start:stop] {
			wg.Add(1)

			go func(kata counterStruct, wg *sync.WaitGroup) {
				defer wg.Done()

				kyu, err := getKataInfo(kata.kataId, cache, &mt2)
				if err != nil {
					return
				}

				mt1.Lock()
				s.CountLangCompleted[kata.language][kyu]++
				mt1.Unlock()
			}(kata, &wg)

		}
		wg.Wait()
		start = stop

	}

	// Перезапись файла cache.json
	if len(pendingRequest) != 0 {
		log.Printf("Update File (%d)\n", len(cache.CacheData))
		if err := cache.UpdateFileCache(); err != nil {
			return errUpdateCache
		}
	}

	return nil
}

// String - строковое представление структуры
func (s *AllInfoUser) String() string {
	result := fmt.Sprintf("%s:\n  Overall rank: %s\n  Points: %d\n  Position in the leaderboard: %d\n  Kata completed: %d\n\n  Statistics:\n",
		s.Username, s.Ranks.Overall.NameRank, s.Honor, s.LeaderboardPosition, s.CodeChallenges.TotalCompleted)

	for lang, ids := range s.LanguagesTotalCompleted {
		result += fmt.Sprintf("    %s (%d)\n", lang, len(ids))

		// Сортировка
		tmp := make([]string, 0)
		for i, _ := range s.CountLangCompleted[lang] {
			tmp = append(tmp, i)
		}
		sort.Strings(tmp)

		for _, kyu := range tmp {
			count := s.CountLangCompleted[lang][kyu]
			result += fmt.Sprintf("      %s - %d\n", kyu, count)
		}
	}
	return result
}

// checkError записывает ошибки в структуру
func (s *AllInfoUser) checkError(err error) {
	s.Err.IsError = true
	s.Err.NameError = err
}

// getKataInfo получает kyu каты по ее ID
func getKataInfo(id string, cache *Cache, mt *sync.Mutex) (string, error) {
	// Отправка запроса
	body, err := getRequest(fmt.Sprintf(kataInfo, id))
	if err != nil {
		return "", err
	}

	// Десерелизация данных
	var kyuR KataKyuStruct
	if err = json.Unmarshal(body, &kyuR); err != nil {
		return "", errGenResponse
	}

	// Проверка пустого значения
	if kyuR.Rank.Name == "" {
		kyuR.Rank.Name = "Beta"
	}

	// Обновления кэша в оперативной памяти
	mt.Lock()
	cache.UpdateCache(id, kyuR.Rank.Name)
	mt.Unlock()

	return kyuR.Rank.Name, nil

}

// getRequest отправляет запрос и возвращает полученные данные
func getRequest(url string) ([]byte, error) {
	// Отправка запроса
	resp, err := http.Get(url)
	if err != nil {
		return nil, badRequest
	}

	// Повторная отправка в случае 429 ошибки
	attempt := time.Duration(2)
	for resp.StatusCode == 429 {
		log.Printf("Stop request (%d seconds). Status code 429 <url %s>", attempt, url)
		time.Sleep(attempt * time.Second)
		attempt *= 2
		resp, err = http.Get(url)
		if err != nil {
			return nil, badRequest
		}
	}

	if resp.StatusCode != 200 {
		return nil, notFound
	}

	// Чтение полученных данных
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errReadData
	}

	return body, nil
}
