# Контракт сервиса OnlineConf

Этот документ описывает реализованный API административного сервиса. Клиент продукта из PR #41 должен быть приведён к этому контракту перед совместным merge. Полное имя для runtime — `scope + "." + key`; в хранилище это два отдельных поля.

## Идентификаторы и JSON

Имена scope и key: `^[a-z][a-z0-9_]{0,63}$`. После создания scope/key неизменяемы. Название `title` и описание редактируемы. Одинаковый key допустим в разных scopes. Ключи конфигов и флагов уникальны внутри своего вида ресурса.

| Назначение | Scope | Key | Значение |
|---|---|---|---|
| Максимальная длина отзыва | `reviews` | `max_length` | Целое число 50..20000 |
| Создание и редактирование отзывов | `reviews` | `write_enabled` | Feature flag |
| Пример настройки интерфейса | `frontend` | `button_color` | JSON-строка, например `"#ff8800"` |

`backend / reviews.max_length` и `backend / reviews.write_enabled` не являются идентификаторами данного контракта. Их нужно заменить в клиенте #41. Проверка лимита и SQL-ограничения сервиса остаются согласованными с исходным контрактом. Миграцию 000016 повторно применять к существующей БД не нужно.

Общие конфиги поддерживают объект, массив, строку, число, boolean и JSON null. Отсутствующий `value` недопустим, явный `value:null` допустим, кроме известных типизированных ключей вроде `reviews/max_length`. Размер JSON value — максимум 64 КиБ в едином нормализованном представлении: без незначащих пробелов и необязательного HTML/Unicode-экранирования, независимо от порядка ключей объектов, с выбором более короткой из точных десятичной и экспоненциальной записей чисел. Проверка использует json.Number, не float64; точность больших чисел сохраняется. Нормализация нужна для измерения и не переписывает сохраняемое значение. Поэтому `&` и `\u0026`, а также `1e3` и `1000` имеют одинаковый размер для лимита. Переданное по HTTP значение может занимать больше 64 КиБ из-за экранирования. HTTP body административного запроса ограничен 256 КиБ; общий snapshot — 4 МиБ фактических байтов ответа, включая экранирование и перевод строки.

Весь сериализованный ответ внутреннего snapshot API ограничен `onlineconf.MaxSnapshotBytes` = 4 МиБ (4 194 304 байта). В предел входят метаданные, все scopes конфигов/флагов, персональные списки, JSON-экранирование и завершающий перевод строки. Ровно 4 МиБ допустимы. Размер проверяется внутри транзакции после сборки новой revision и до commit. Превышение возвращает HTTP 413 с code `snapshot_too_large`; изменение ресурса, версия, аудит и revision откатываются вместе. Параллельные записи проверяют общий бюджет последовательно под блокировкой oc_state.

При загрузке и перед публикацией кеш также отклоняет превышающий предел snapshot и сохраняет последнюю рабочую версию. Потребитель должен использовать тот же `MaxSnapshotBytes`, а не отдельный отличающийся лимит.

Продуктовый fallback: `reviews.max_length=20000`, `reviews.write_enabled=false`. Сбой доставки сохраняет последний успешный snapshot. Любой новый ключ начинает влиять на продукт после подключения его использования в продуктовом коде.

## Общие Go-пакеты

- `internal/domain/onlineconf`: `SnapshotPayload`, `SnapshotConfig`, `SnapshotFlag`, доменные ошибки и проверки.
- `internal/services/onlineconfevaluator`: единственная функция `Evaluate` для административной проверки и продуктового бэкенда; без зависимостей от SQL, Redis, HTTP и административных use cases.
- `internal/delivery/middleware`: общие request ID, logging и recover. Он не импортирует продуктовый или административный router.

Административный HTTP-тип называется `AdminHandler`. Серверные настройки — `config.OnlineConfServerConfig`, загрузчик — `LoadOnlineConfServer`. Это исключает пересечение с клиентскими `config.OnlineConf` и `LoadOnlineConf` из #41. Go-константа серверного токена называется `EnvOnlineConfServerServiceToken`; имя переменной окружения осталось `ONLINECONF_SERVICE_TOKEN`.

Кеш административного сервиса находится в `internal/services/onlineconf/snapshot_store.go`. Продуктовый клиент и кеш рекомендуется разместить в `internal/services/onlineconfruntime`, а HTTP runtime — в `internal/delivery/onlineconfruntime`.

## Внутренний snapshot API

`GET /internal/onlineconf/snapshot`

Требуется `Authorization: Bearer <ONLINECONF_SERVICE_TOKEN>`. Пользовательская cookie и заголовок с ролью не заменяют сервисную авторизацию. Участник 2 не должен проксировать этот endpoint наружу.

```json
{
  "revision": 42,
  "generated_at": "2026-11-28T09:00:00Z",
  "configs": [
    {"scope":"reviews","key":"max_length","value":500},
    {"scope":"frontend","key":"button_color","value":"#ff8800"}
  ],
  "flags": [
    {
      "id":1,
      "scope":"reviews",
      "key":"write_enabled",
      "enabled":true,
      "rollout_percent":25,
      "salt":"a1b2",
      "force_on_user_ids":[201],
      "force_off_user_ids":[202]
    }
  ]
}
```

`revision` — целое число >= 0. Пустая БД возвращает revision 0 и пустые массивы configs/flags. `generated_at` — время RFC 3339 в UTC; оно не заменяет revision. Все конфиги и флаги принадлежат одной согласованной ревизии. Ответ не содержит пагинации, title или административных версий.

Клиент может использовать `json.Decoder.DisallowUnknownFields`, если декодирует именно `onlineconf.SnapshotPayload`. После декодирования необходимо проверить конец JSON и вызвать `Validate()`. Числа в произвольном конфиге хранятся в `json.RawMessage`, не преобразуются автоматически во float64. Персональные списки должны содержать положительные безопасные целые JS и не пересекаться.

Для evaluator транспортный флаг преобразуется через `SnapshotFlag.AsFeatureFlag()`. Преобразование копирует персональные списки. Флаг не содержит title/version: для вычисления они не нужны; `ValidateRules()` проверяет правила отдельно от административных полей.

Фикстуры для клиента: `tests/onlineconf/testdata/snapshot_empty.json` и `snapshot_full.json`.

## Вычисление флага

Общий пакет: `internal/services/onlineconfevaluator`, функция `Evaluate(flag, userID)`.

1. Нет флага, нет пользователя или enabled=false: false.
2. Пользователь в force-off: false.
3. Пользователь в force-on: true.
4. Иначе применяется процент.

Глобальное выключение сильнее force-on. На повреждённых данных пересечение списков разрешается в пользу force-off; административный API такое изменение отклоняет.

```text
input = salt + ":" + decimal_user_id
hash = SHA256(UTF8(input))
bucket = uint64_big_endian(hash[0:8]) % 10000
enabled = bucket < rollout_percent * 100
```

Соль генерируется сервером при создании и сохраняется при изменении. Title, ключ и revision в хеш не входят. При неизменных salt/overrides/enabled рост процента не исключает уже включённых пользователей.

Причины: `globally_disabled`, `forced_off`, `forced_on`, `rollout_match`, `rollout_miss`. Bucket равен null до процентного правила и целому 0..9999 после него.

| Salt | User ID | Bucket | Результат при 25%, без overrides |
|---|---:|---:|---|
| a1b2 | 1 | 4554 | false |
| a1b2 | 2 | 6640 | false |
| a1b2 | 101 | 1301 | true |

## Административный API

База: `/api/onlineconf/admin`. Все операции, включая чтение, требуют cookie `session_id` и актуальную серверную роль admin. Actor берётся из проверенной сессии. Изменяющие запросы защищены `http.CrossOriginProtection`; cookie-настройки основного продукта не меняются.

| Метод | Путь | Назначение |
|---|---|---|
| GET | /me | ID, username, роль текущего администратора |
| GET, POST | /scopes | Список / создание scope |
| GET, PUT, DELETE | /scopes/{id} | Карточка / изменение / удаление пустого scope |
| GET, POST | /configs | Список / создание конфигурации |
| GET, PUT, DELETE | /configs/{id} | Карточка / изменение / удаление |
| GET, POST | /flags | Список / создание флага |
| GET, PUT, DELETE | /flags/{id} | Карточка / изменение / удаление |
| POST | /flags/{id}/evaluate | Проверка заданного пользователя |
| GET | /audit | История |

Поиск списков: `scope`, `q` (подстрока key/title без учёта регистра), `limit` (1..100, по умолчанию 50), `offset` (>=0). Ответ: `{ "items": [], "total": 0 }`. `%` и `_` в q не являются SQL wildcard.

POST scope: `{ "name":"reviews", "title":"Отзывы" }`.

POST config:

```json
{"scope":"reviews","key":"max_length","title":"Лимит","description":"","value":500}
```

POST flag:

```json
{"scope":"reviews","key":"write_enabled","title":"Отзывы","description":"","enabled":true,"rollout_percent":25,"force_on_user_ids":[],"force_off_user_ids":[]}
```

Создание возвращает HTTP 201 и version=1. PUT передаёт те же поля, обязательно description и текущую version; отсутствующие обязательные поля не заменяются значениями по умолчанию. Административный ответ флага не содержит salt. Клиент не может передавать actor_id или salt.

DELETE передаёт `?version=N`, успех — 204. Устаревшая версия возвращает 409 и ничего не меняет. Непустой scope удалить нельзя. Изменение ресурса, аудит и сохранение snapshot/revision выполняются одной транзакцией.

Проверка пользователя: POST `/flags/{id}/evaluate` с `{ "user_id":101 }`. Ответ: user_id, enabled, reason, bucket, revision. Несуществующий пользователь/флаг — 404. Для решения используется один snapshot, revision относится к использованным правилам.

История фильтруется по resource_type (`scope`, `config`, `flag`), resource_id, limit, offset. Запись содержит actor_id, actor_name, resource_type, resource_id, scope/key, action, before, after, revision и created_at UTC. При создании before=null, при удалении after=null. История переживает удаление ресурса и не содержит соль флага.

Ошибка: `{ "error":"...", "code":"..." }`. Основные коды: 400 invalid_input/invalid_json, 401 unauthorized, 403 forbidden, 404 not_found, 409 version_conflict/duplicate_key/resource_in_use, 405 method_not_allowed, 413 snapshot_too_large, 500 internal_error, 503 runtime_not_ready.

## Обновление и публичный runtime

OnlineConf обновляет кеш после commit и каждые 10 секунд. У него один worker, таймаут загрузки — 2 секунды. Меньшая revision не заменяет большую. При ошибке сохраняется последний успешный snapshot. Холодный старт без доступа к snapshot завершается ошибкой; процесс не объявляет готовность.

Продуктовый кеш обновляется каждые 10 секунд, фронтенд — каждые 15 секунд. Номинальный бюджет 10+10+15=35 секунд; на работающей сети суммарная задержка должна быть меньше минуты.

Продуктовый runtime реализует участник 2. Конфиги в нём должны поддерживать любой JSON (`map[string]json.RawMessage`), флаги — вычисленные bool для пользователя из серверной сессии. Нужен явный список публикуемых ключей, а не публикация всего snapshot. Salt, overrides, сервисные токены и внутренние конфиги в браузер не попадают. Заголовок runtime: `Cache-Control: private, no-store`.

## Сборка и передача участнику 2

```bash
docker build -f Dockerfile.onlineconf -t onlyseans_onlineconf:local .
```

Образ запускает `/onlineconf`, собираемый из `./cmd/onlineconf`. Runtime работает от UID/GID 65532, пишет логи в stdout, не содержит исходники и инструменты сборки. Healthcheck встроен: `/onlineconf -healthcheck` проверяет `/healthz` на настроенном порту. Порт по умолчанию 8081.

Обязательное окружение: DATABASE_URL, REDIS_URL, ONLINECONF_SERVICE_TOKEN. Дополнительно: ONLINECONF_HOST, ONLINECONF_PORT, ONLINECONF_ALLOWED_ORIGINS (через запятую), LOG_LEVEL, LOG_FORMAT. Секреты в образ не записываются.

Участнику 2 нужно подключить в compose:

```yaml
onlineconf:
  image: onlyseans_onlineconf:local
  build:
    context: .
    dockerfile: Dockerfile.onlineconf
  environment:
    DATABASE_URL: ${DATABASE_URL}
    REDIS_URL: redis://redis:6379
    ONLINECONF_SERVICE_TOKEN: ${ONLINECONF_SERVICE_TOKEN}
```

Адреса должны соответствовать compose-сети и существующему пользователю БД. Миграция 000016 применяется отдельно к существующему volume; docker-entrypoint-initdb.d повторно не запускается. Административная SPA и продуктовый фронтенд поставляются отдельными участниками.

Перед совместным merge нужны общая целевая ветка, сборка обоих PR вместе, адаптация клиента #41 и сквозная проверка реального HTTP-ответа этим клиентом. Тесты данного PR не означают, что неизменённый клиент #41 уже совместим.

Для обновления существующей БД данные не переписываются: регрессионный тест проверяет загрузку старого массива из 17 000 единиц и строк с Unicode-разделителями. Временные схемы PostgreSQL-тестов имеют случайные 128-битные имена, независимые от часов и других процессов.
