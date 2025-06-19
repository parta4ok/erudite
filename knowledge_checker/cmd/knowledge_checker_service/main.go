package main

import (
)

func main() {

	// _ = godotenv.Load()

	// connStr := os.Getenv("TESTPGCONN")
	// fmt.Println(connStr)

	// pg, err := postgres.NewStorage(connStr)
	// if err != nil {
	// 	panic(err)
	// }

	// q, err := pg.GetQuesions(context.TODO(), []string{"Базы данных"})
	// if err != nil {
	// 	panic(err)
	// }

	// session, err := entities.NewSession(1, []string{"Базы данных"}, generator.NewUint64Generator())
	// if err != nil {
	// 	panic(err)
	// }

	// if err := pg.StoreSession(context.TODO(), session); err != nil {
	// 	panic(err)
	// }

	// questionsMap := make(map[uint64]entities.Question, len(q))
	// for _, question := range q {
	// 	questionsMap[question.ID()] = question
	// }

	// if err := session.SetQuestions(questionsMap, time.Minute*5); err != nil {
	// 	panic(err)
	// }

	// if err := pg.StoreSession(context.TODO(), session); err != nil {
	// 	panic(err)
	// }

	// answers := make([]*entities.UserAnswer, 0, len(questionsMap))

	// for id, question := range questionsMap {
	// 	answer, err := entities.NewUserAnswer(id, []string{question.Variants()[0]})
	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	answers = append(answers, answer)
	// }

	// if err := session.SetUserAnswer(answers); err != nil {
	// 	panic(err)
	// }

	// if err := pg.StoreSession(context.TODO(), session); err != nil {
	// 	panic(err)
	// }

	// fmt.Println(session.GetStatus())

	// recoveredSession, err := pg.GetSessionBySessionID(context.TODO(), session.GetSesionID())
	// if err != nil {
	// 	panic(err)
	// }
	// res, _ := session.GetSessionResult()
	// fmt.Println(recoveredSession.GetStatus(), session.GetTopics(), res)
}
