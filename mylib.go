package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

type Person struct {
	first_name  string
	family_name string
	birth_year  int
	death_year  int
}

type Writer struct {
	Person
}

func (w Writer) getFullName() string {
	return fmt.Sprintf("%s %s", w.first_name, w.family_name)
}

func (w Writer) getLifeYears() string {
	return fmt.Sprintf("%v - %v", w.birth_year, w.death_year)
}

type Book struct {
	name         string
	writer       Writer
	publish_year int
}

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "password"
	dbname   = "mylib"
)

func main() {
	psql_info := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", psql_info)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Successfully connected!")
}

/*
1. Поиск писателя по имени
	- показать информацию о писателе
	- показать книги писателя
2. Поиск книги по названию
	- показать информацию о книге (писатель)
3. Добавить писателя
4. Добавить книгу (с уже имеющимся писателем или новым)
5. Добавить книгу как прочитанную (дата, оценка)
6. Посмотреть список всех прочитанных книг (сортировка по дате, по алфавиту, по оценке)
7. Список "Избранное", что хочется прочитать
8. Список любимые писатели
*/
