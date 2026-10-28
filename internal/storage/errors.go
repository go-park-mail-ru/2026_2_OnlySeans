package storage

const (
	UniqueViolation     = "23505" // нарушена уникальность, у автора уже есть отзыв на этот фильм
	ForeignKeyViolation = "23503" // нарушена внешняя ссылка, фильма с таким номером нет в базе

)
