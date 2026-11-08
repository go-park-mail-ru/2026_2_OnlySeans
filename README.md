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

### Порты Docker Compose

Redis доступен с компьютера на порту `8020`, S3 API — на `8050`,
консоль RustFS — на `8051`. Внутри Docker backend использует адреса
`redis://redis:6379` и `http://rustfs:9000`.

Для запуска на виртуалке укажите в `.env` адрес S3, доступный браузеру:

```dotenv
S3_PUBLIC_ENDPOINT=http://IP_ВИРТУАЛКИ:8050
```

Замените `IP_ВИРТУАЛКИ` на реальный IP или домен. При локальном запуске
Compose использует `http://localhost:8050` по умолчанию.

После изменения портов пересоздайте контейнеры:

```bash
docker compose up -d --build
```
