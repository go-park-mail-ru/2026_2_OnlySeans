# 2026_2_OnlySeans 
Репозиторий команды OnlySeans. Проект Кинопоиск

### [Деплой фронтенда]()  
### [Деплой бекенда]()  
### [Репозиторий фронтэнда](https://github.com/frontend-park-mail-ru/2026_2_OnlySeans)  
### [Макет]()  


### Состав команды:
- [Яковлева Светлана](https://github.com/Svetlana17yak)
- [Ушакова Полина](https://github.com/polinauss)
- [Городов Егор](https://github.com/Pom1dorka-Salat1k)
- [Нестеров Георгий](https://github.com/GeR017)

### Менторы:
- [Шахмар Абдуллаев](https://github.com/abdullShah) - Frontend
- [Антон Самойлов](https://github.com/samantonio28) - Backend
- [???]() - UX
- [???]() - Database

### Интеграционные тесты

`make test` также компилирует интеграционные тесты, чтобы устаревшие импорты
не оставались незамеченными. Этот шаг не обращается к PostgreSQL и S3.

Для реального прогона выполните `make test-integration`, передав в окружении:

- `POSTERS_TEST_DATABASE_URL`, `AVATAR_TEST_DATABASE_URL`, `PERSONS_TEST_DATABASE_URL`, `MIGRATIONS_TEST_DATABASE_URL` — DSN PostgreSQL;
- `INTEGRATION_S3_ENDPOINT` — доступный с хоста S3 endpoint, например `http://localhost:8050`;
- `INTEGRATION_S3_BUCKET` — существующий тестовый bucket;
- `RUSTFS_ACCESS_KEY`, `RUSTFS_SECRET_KEY` — ключи доступа к этому хранилищу.

Цель завершается ошибкой, если конфигурация отсутствует. SQL-проверки используют
временные таблицы одной сессии, HTTP-серверы и gRPC-транспорт создаются внутри
тестов. Проверки доступа не повышают роли существующих пользователей. В S3
создаются только объекты со случайными именами; тесты удаляют свои объекты при
завершении, включая завершение с ошибкой. Рабочие таблицы и существующие объекты
хранилища не изменяются.
