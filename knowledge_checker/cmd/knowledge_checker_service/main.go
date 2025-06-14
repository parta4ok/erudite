package main

import (
	"context"
	"fmt"
	"time"

	"github.com/parta4ok/kvs/knowledge_checker/internal/adapter/generator"
	"github.com/parta4ok/kvs/knowledge_checker/internal/adapter/storage/postgres"
	"github.com/parta4ok/kvs/knowledge_checker/internal/entities"
)

func main() {
	pg, err := postgres.NewStorage("postgresql://postgres:password@localhost:5432/knowledge?sslmode=disable")
	if err != nil {
		panic(err)
	}

	q, err := pg.GetQuesions(context.TODO(), []string{"Базы данных"})
	if err != nil {
		panic(err)
	}

	session, err := entities.NewSession(1, []string{"Базы данных"}, generator.NewUint64Generator())
	if err != nil {
		panic(err)
	}

	if err := pg.StoreSession(context.TODO(), session); err != nil {
		panic(err)
	}

	questionsMap := make(map[uint64]entities.Question, len(q))
	for _, question := range q {
		questionsMap[question.ID()] = question
	}

	if err := session.SetQuestions(questionsMap, time.Millisecond*200); err != nil {
		panic(err)
	}

	if err := pg.StoreSession(context.TODO(), session); err != nil {
		panic(err)
	}

	answers := make([]*entities.UserAnswer, 0, len(questionsMap))

	for id, question := range questionsMap {
		answer, err := entities.NewUserAnswer(id, []string{question.Variants()[0]})
		if err != nil {
			panic(err)
		}

		answers = append(answers, answer)
	}
	time.Sleep(time.Second)
	if err := session.SetUserAnswer(answers); err != nil {
		panic(err)
	}

	if err := pg.StoreSession(context.TODO(), session); err != nil {
		panic(err)
	}

	fmt.Println(session.GetSessionResult())

	// println(q)
}
