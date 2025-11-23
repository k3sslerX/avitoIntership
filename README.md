# Тестовое задание для стажёра Backend (Avito)

- [Сборка и запуск](#сборка-и-запуск)  
  - [Локально](#локально)
  - [Сборка Docker-контейнера](#сборка-docker-контейнера)
  - [Использование готового Docker-образа](#использование-готового-docker-изображения-из-ghcrio-)
- [Детали реализации](#детали-реализации)  
  - [ER-модель базы данных](#er-модель-базы-данных) 
  - [Сбор статистики](#сбор-статистики)
- [Нагрузочное тестирование](#нагрузочное-тестирование)
- [Проблемы](#проблемы)
- [Возможные улучшения](#возможные-улучшения)

## Сборка и запуск

### Локально
```bash
make build
```
Также требуется задать переменные окружения в файле `.env` по примеру из `.env.example`

### Сборка Docker-контейнера
```bash
make build-image
```
или сборка с запуском
```bash
make run-container
```

Там же поднимается PostgreSQL, нужно убедиться что порт `5432` в данный момент не прослушивается другим процессом

### Использование готового Docker-изображения из *ghcr.io* \*
```bash
docker pull ghcr.io/k3sslerx/avitointership/prreviewersautoassigner:23607ac8bc1a7148f6e614ed66e7b5fcee260e82
```
*\* Требуется установленный и запущенный PostgreSQL и созданная в нём база данных (согласно `.env`)*

## Детали реализации

### ER-модель базы данных
![](https://github.com/k3sslerX/avitoIntership/blob/main/images/ER-модель.png)

Будем считать, что один пользователь может состоять только в одной команде.

### Сбор статистики

Эндпоинт получения статистики:

`GET /stats`

## Нагрузочное тестирование

### Создание Pull Request-ов
```azure
Concurrency Level:      5
Time taken for tests:   0.292 seconds
Complete requests:      50
Failed requests:        0
Non-2xx responses:      50
Total transferred:      8900 bytes
Total body sent:        12950
HTML transferred:       3200 bytes
Requests per second:    171.50 [#/sec] (mean)
Time per request:       29.154 [ms] (mean)
Time per request:       5.831 [ms] (mean, across all concurrent requests)
Transfer rate:          29.81 [Kbytes/sec] received
                        43.38 kb/s sent
                        73.19 kb/s total

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    0   0.0      0       0
Processing:    16   27   5.6     26      44
Waiting:       16   27   5.6     26      44
Total:         16   27   5.6     26      44
```

### Получение информации о Pull Request-е
```azure
Concurrency Level:      10
Time taken for tests:   0.018 seconds
Complete requests:      100
Failed requests:        0
Non-2xx responses:      100
Total transferred:      17600 bytes
HTML transferred:       1900 bytes
Requests per second:    5496.02 [#/sec] (mean)
Time per request:       1.819 [ms] (mean)
Time per request:       0.182 [ms] (mean, across all concurrent requests)
Transfer rate:          944.63 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    0   0.0      0       0
Processing:     1    2   0.4      1       3
Waiting:        1    2   0.4      1       3
Total:          1    2   0.4      2       3
```

### Получение статистики
```azure
Concurrency Level:      20
Time taken for tests:   1.964 seconds
Complete requests:      200
Failed requests:        0
Total transferred:      104000 bytes
HTML transferred:       82200 bytes
Requests per second:    101.83 [#/sec] (mean)
Time per request:       196.397 [ms] (mean)
Time per request:       9.820 [ms] (mean, across all concurrent requests)
Transfer rate:          51.71 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    0   0.1      0       1
Processing:    37  189  48.3    183     307
Waiting:       37  189  48.3    183     307
Total:         38  189  48.3    184     307
```

### Получение информации о команде
```azure
Concurrency Level:      10
Time taken for tests:   0.158 seconds
Complete requests:      100
Failed requests:        0
Total transferred:      34700 bytes
HTML transferred:       23800 bytes
Requests per second:    633.40 [#/sec] (mean)
Time per request:       15.788 [ms] (mean)
Time per request:       1.579 [ms] (mean, across all concurrent requests)
Transfer rate:          214.64 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    0   0.1      0       1
Processing:     8   15   2.9     15      23
Waiting:        7   14   2.9     15      23
Total:          8   15   2.9     15      23
```

Исходя из результатов проведённого тестирования, в SLI времени ответа не укладываются только некоторые запросы статистики

## Проблемы

 - Интеграционное тестирование
   - Не рассчитал своё время и просто не успел его сделать
 - Конфликт портов
   - При поднятом PostgreSQL контейнер с базой данных не запустится, а контейнер с сервисом (по крайней мере на моей машине) к локальной базе данных подключаться не хочет
 - Миграции
   - Очень долго не мог понять, почему не инициализируются миграции (честно, до сих пор не понимаю почему и как они работают)

## Возможные улучшения

 - Кэширование статистики
   - Последние обращения к статистике неплохо было бы кешировать с использованием Redis.
 - Перестройка базы данных
   - Сейчас один пользователь может быть только частью одной команды. При удалении команды удаляется и пользователь. Изначально база данных спроектирована некорректно, было бы неплохо исправить.
 - Интеграция с GitHub/GitLab
   - Подключить сервис напрямую к GitHub/GitLab, чтобы ревьюер сразу назначался непосредственно в самом репозитории. На данный момент этот сервис работает только со своей базой данных.
