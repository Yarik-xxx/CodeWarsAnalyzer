package cwapi

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

type Cache struct {
	CacheData map[string]string
}

// Init инициализирует кэш рангов кат из файла cache.json
func (c *Cache) Init() error {
	if _, err := os.Stat("cache/cache.json"); os.IsNotExist(err) {
		log.Printf("Cache file not found. Create a new file:\n")
		log.Printf("...creating a \"cache\" folder\n")
		if err := os.Mkdir("cache", 0777); err != nil {
			return err
		}

		log.Printf("...creating a \"cache.json\" file\n")
		f, err := os.Create("cache/cache.json")
		defer f.Close()
		if err != nil {
			return err
		}

		_, err = f.Write([]byte("{}"))

		if err != nil {
			return err
		}
		log.Printf("...ready\n")
	}

	f, err := os.Open("cache/cache.json")
	defer f.Close()
	if err != nil {
		return err
	}
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &c.CacheData)
	if err != nil {
		return err
	}
	return nil
}

// UpdateCache обновляет кэш в оперативной памяти
func (c *Cache) UpdateCache(id string, kyu string) {
	c.CacheData[id] = kyu
}

// UpdateFileCache обновляет файл с кэшем cache.json
func (c *Cache) UpdateFileCache() error {
	writeData, err := json.Marshal(c.CacheData)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("cache/cache.json", writeData, 0)
	if err != nil {
		return err
	}
	return nil
}
