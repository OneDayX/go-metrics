# go-musthave-metrics-tpl

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

## Статический анализ

В `cmd/staticlint` лежит multichecker, код проекта должен проходить его без замечаний:

```
make lint                      # собрать cmd/staticlint/staticlint и проверить ./...
go run ./cmd/staticlint ./...  # то же без make
```

В него входят стандартные анализаторы `golang.org/x/tools/go/analysis/passes`, все проверки класса SA из staticcheck, несколько проверок классов S, ST и QF, публичные `errcheck` и `bodyclose` и собственный `osexit`, который запрещает прямой вызов `os.Exit` в функции `main` пакета `main`. Запуск, флаги и назначение каждого анализатора описаны в документации пакета: `go doc ./cmd/staticlint`.

## Генерация Reset

`cmd/reset` ищет во всех пакетах проекта структуры с комментарием `// generate:reset` и пишет для них метод `Reset()` в файл `reset.gen.go` того же пакета:

```
make generate        # то же, что go run ./cmd/reset
```

`Reset()` обнуляет примитивы, обрезает слайсы до `[:0]`, очищает мапы, вызывает `Reset()` у вложенных полей, у которых он есть, и сбрасывает значения не nil указателей по тем же правилам. Пример входа и результата: `cmd/reset/testdata`.

## Покрытие тестами

```
make cover
```

## Бенчмарки

Бенчмарки лежат рядом с кодом в файлах `*_bench_test.go`: хранилище в памяти (`Update`, `UpdateBatch`, `FetchAll`), хендлеры (`/updates/`, `/value/`, `/`), gzip middleware, подпись (`sign.Sum`), сжатие на агенте (`compress.Encode`) и сбор метрик агентом (`Collect`).

```
make bench
```

## Профилирование памяти

Сервер умеет отдавать профили `net/http/pprof` на отдельном адресе: флаг `-pprof` или переменная `PPROF_ADDRESS` (например, `localhost:6060`). По умолчанию pprof выключен.

Профили снимаются командой `make profile` (нужен [hey](https://github.com/rakyll/hey): `go install github.com/rakyll/hey@latest`). Она собирает сервер, запускает его с хранением в памяти и даёт нагрузку, по 20 000 запросов в 50 потоков:

- `POST /updates/` — сжатый gzip батч из 35 метрик, как у агента (`profiles/testdata/updates.json`);
- `POST /value/` — запрос одной метрики (`profiles/testdata/value.json`);
- `GET /` — HTML-страница со всеми метриками.

Сразу после нагрузки сохраняется профиль `/debug/pprof/allocs`:

```
make profile                 # profiles/base.pprof
make profile PROFILE=result  # profiles/result.pprof
```

Это тот же heap-профиль, но по умолчанию в нём показывается `alloc_space` — вся память, выделенная за время нагрузки. Нагрузка в обоих прогонах одинаковая, поэтому разница показывает, сколько памяти стало нужно на ту же работу.

### Что нашлось в профиле

| Место | Было | Причина | Исправление |
|---|---|---|---|
| `compress/flate.NewWriter` | 30,8 ГБ (93%) | gzip middleware создавал новый `gzip.Writer` на каждый ответ, а он при первой записи выделяет около 800 КБ | `sync.Pool` для `gzip.Writer` и `Reset` перед каждым ответом |
| `compress/gzip.NewReader` | 832 МБ | новый распаковщик на каждый сжатый запрос | `sync.Pool` для `gzip.Reader` |
| `html/template` Parse и escape | 244 МБ | шаблон страницы `/` разбирался на каждый запрос | шаблон разбирается один раз при старте |

### Результат

Всего выделено памяти за нагрузку: **33 039 МБ → 993 МБ**.

| Бенчмарк | Было | Стало |
|---|---|---|
| `BenchmarkGzipMiddleware` | 147 µs, 867 КБ, 59 allocs | 17 µs, 15 КБ, 36 allocs |
| `BenchmarkList` | 54 µs, 38,8 КБ, 572 allocs | 39 µs, 24,4 КБ, 452 allocs |

Запросов в секунду (hey): `/updates/` 26 000 → 44 000, `/value/` 14 000 → 82 000, `/` 7 900 → 29 000.

Вывод `pprof -top -diff_base=profiles/base.pprof profiles/result.pprof` (через `go tool pprof`):

```
File: server
Build ID: fdfff6bbff02d3f8ff98faf48cf13e985a312865
Type: alloc_space
Time: 2026-10-13 19:01:48 +05
Showing nodes accounting for -31512.43MB, 95.38% of 33039.21MB total
Dropped 242 nodes (cum <= 165.20MB)
      flat  flat%   sum%        cum   cum%
-25276.82MB 76.51% 76.51% -30701.54MB 92.92%  compress/flate.NewWriter (inline)
-5278.61MB 15.98% 92.48% -5278.61MB 15.98%  compress/flate.(*compressor).initDeflate (inline)
 -552.92MB  1.67% 94.16%  -552.92MB  1.67%  compress/flate.(*dictDecoder).init (inline)
 -250.05MB  0.76% 94.91%  -250.05MB  0.76%  compress/flate.(*huffmanEncoder).generate
 -138.52MB  0.42% 95.33%  -691.44MB  2.09%  compress/flate.NewReader
  -12.01MB 0.036% 95.37%  -832.34MB  2.52%  compress/gzip.NewReader (inline)
      -3MB 0.0091% 95.38% -31040.23MB 93.95%  main.run.Logger.func3.1
      -1MB 0.003% 95.38% -15356.39MB 46.48%  main.run.(*Handler).ValueJSON.func11
       1MB 0.003% 95.38% -15397.65MB 46.60%  text/template.(*Template).execute
    0.50MB 0.0015% 95.38%  -718.52MB  2.17%  github.com/OneDayX/go-metrics/internal/server/middleware.newGzipReader
   -0.50MB 0.0015% 95.38% -15670.80MB 47.43%  main.run.(*Handler).List.func6
   -0.50MB 0.0015% 95.38% -32045.92MB 96.99%  net/http.(*conn).serve
         0     0% 95.38%  -250.05MB  0.76%  compress/flate.(*Writer).Close (inline)
         0     0% 95.38%  -250.05MB  0.76%  compress/flate.(*compressor).close
         0     0% 95.38%  -250.05MB  0.76%  compress/flate.(*compressor).deflate
         0     0% 95.38% -5424.72MB 16.42%  compress/flate.(*compressor).init
         0     0% 95.38%  -250.05MB  0.76%  compress/flate.(*compressor).writeBlock
         0     0% 95.38%  -168.37MB  0.51%  compress/flate.(*huffmanBitWriter).indexTokens
         0     0% 95.38%  -250.05MB  0.76%  compress/flate.(*huffmanBitWriter).writeBlock
         0     0% 95.38%  -707.01MB  2.14%  compress/gzip.(*Reader).Reset
         0     0% 95.38%  -691.44MB  2.09%  compress/gzip.(*Reader).readHeader
         0     0% 95.38%  -250.05MB  0.76%  compress/gzip.(*Writer).Close
         0     0% 95.38% -30701.54MB 92.92%  compress/gzip.(*Writer).Write
         0     0% 95.38% -15348.89MB 46.46%  encoding/json.(*Encoder).Encode
         0     0% 95.38%  -249.55MB  0.76%  github.com/OneDayX/go-metrics/internal/server/middleware.(*gzipResponseWriter).Close
         0     0% 95.38% -30708.54MB 92.95%  github.com/OneDayX/go-metrics/internal/server/middleware.(*gzipResponseWriter).Write
         0     0% 95.38% -30708.54MB 92.95%  github.com/OneDayX/go-metrics/internal/server/middleware.(*responseWriter).Write
         0     0% 95.38% -32011.30MB 96.89%  github.com/OneDayX/go-metrics/internal/server/middleware.GzipMiddleware.func1
         0     0% 95.38%  -719.52MB  2.18%  github.com/OneDayX/go-metrics/internal/server/middleware.decompressRequest
         0     0% 95.38% -32011.80MB 96.89%  github.com/go-chi/chi/v5.(*Mux).ServeHTTP
         0     0% 95.38% -31035.21MB 93.93%  github.com/go-chi/chi/v5.(*Mux).routeHTTP
         0     0% 95.38% -31035.21MB 93.93%  github.com/go-chi/chi/v5/middleware.StripSlashes.func1
         0     0% 95.38% -15562.76MB 47.10%  html/template.(*Template).Execute
         0     0% 95.38%  -165.11MB   0.5%  html/template.(*Template).escape
         0     0% 95.38%  -165.11MB   0.5%  html/template.escapeTemplate
         0     0% 95.38% -32011.30MB 96.89%  net/http.HandlerFunc.ServeHTTP
         0     0% 95.38% -32011.80MB 96.89%  net/http.serverHandler.ServeHTTP
         0     0% 95.38% -15397.65MB 46.60%  text/template.(*Template).Execute (inline)
         0     0% 95.38% -15398.65MB 46.61%  text/template.(*state).walk
```
