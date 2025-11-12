# vk-sender

## Что делает

1. Подписывается на Kafka-топик `TOPIC_NAME_VK_REQUEST_MESSAGE` в группе `GROUP_ID_VK_SENDER`.
2. Каждое сообщение десериализует в `contract.SendMessageRequest`, подготавливает параметры `peer_id`, `message`, `random_id` и `access_token`.
3. Отправляет POST-запрос к `https://api.vk.com/method/messages.send` с `application/x-www-form-urlencoded`.
4. Возвращает ошибку, если VK API ответил статусом ≥300 или вернул JSON с `error`, чтобы `runConsumerSupervisor` мог перезапустить поток.

## Запуск

1. Установи переменные окружения (например, `set -a && source .env && set +a`).
2. Собери и запусти локально:
   ```bash
   go run ./cmd/vk-sender
   ```
3. Либо собери Docker-образ и запусти его:
   ```bash
   docker build -t vk-sender .
   docker run --rm -e ... vk-sender
   ```

## Переменные окружения

Все обязательны, кроме SASL, если Kafka открыта.

- `KAFKA_BOOTSTRAP_SERVERS_VALUE` — список брокеров `host:port[,host:port]`.
- `KAFKA_TOPIC_NAME_VK_REQUEST_MESSAGE` — топик для `SendMessageRequest`, откуда читается consumer.
- `KAFKA_GROUP_ID_VK_SENDER` — consumer group id.
- `KAFKA_CLIENT_ID_VK_SENDER` — client id (producer+consumer) для метрик.
- `KAFKA_SASL_USERNAME` и `KAFKA_SASL_PASSWORD` — если Kafka требует SASL/SCRAM.
- `VK_TOKEN` — токен сообщества/бота для `access_token` при вызове API.

## Примечания

- `random_id` генерируется через `crypto/rand` (см. `internal/processor/vk-message-sender.go`), чтобы VK API не отвергнул дубликаты.
- Ответы VK API парсятся в `vkAPIResponse`, и при наличии `error` сервис возвращает ошибку.
- Сообщения отправляются синхронно, логируются (peer_id и статус) и в случае неудачи переотправляются через supervisor.
