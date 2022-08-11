package cwapi

import "errors"

// Ссылки
const (
	baseInfo          = "https://www.codewars.com/api/v1/users/%s"
	listCompletedKata = "https://www.codewars.com/api/v1/users/%s/code-challenges/completed?page=%d"
	kataInfo          = "https://www.codewars.com/api/v1/code-challenges/%s"
	chunk             = 5 // максимальное количество горутин
)

// Ошибки
var (
	notFound       = errors.New("user is not found")
	badRequest     = errors.New("error sending request to codewars.com")
	errReadData    = errors.New("an error occurred while reading the response data")
	errGenResponse = errors.New("an error occurred while generating the response")
	errUpdateCache = errors.New("an error occurred while updating the cache")
)

type AllInfoUser struct {
	LanguagesTotalCompleted map[string][]struct {
		Id   string
		Name string
	}
	CountLangCompleted map[string]map[string]int
	Ranks              struct {
		Overall   ObjectRank
		Languages map[string]ObjectRank
	}
	Username            string
	Honor               int
	LeaderboardPosition int
	CodeChallenges      struct {
		TotalAuthored  int
		TotalCompleted int
	}
	Err struct {
		IsError   bool
		NameError error
	}
}

type ObjectRank struct {
	NameRank string `json:"name"`
	Score    int
}

type CompletedKataStruct struct {
	Data []struct {
		CompletedLanguages []string
		Id                 string
		Name               string
	}
}

type KataKyuStruct struct {
	Rank struct {
		Name string
	}
}

type responseByte struct {
	data []byte
	err  error
}

type counterStruct struct {
	language string
	kataId   string
}
