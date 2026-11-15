# B25 freeze и отдельный source CI WIP

Root END17:19 принят: config35fd4c11 + Linux992dab86/runtimeab0511ff, layout12/result34 PASS. Root summary04d69bd32f880e2ce821d1045619d9416f7add188e09737ed50de57b5ea3f411, cleanupdf3802cc48d5d2d00c6ad79fd0ed8502a950181e8fe03f9330cae1284acffb01. B25 выбран для3 баз, B24 fallback. Shared не переключал; исполнение выпуска только её словом.

Owner B25: sourcea2ba78f8174c0bc26e5724b293408de25047a323/runtimeab0511ff927b29475af07b80501a71a4519e1bc7, finalmanifestfa4a4577d92f224f17093917e5ed771ea1b2ec48440e30c71951377b10c15c34. PG230 PASS, Windows native24 PASS/JS0/cleanup; Linux canonical root отдельный. Readiness1730 и finalmanifest ранее переданы.

Source CI NOT GREEN: lint показал4 unchecked Close в новых тестах. Они исправлены только в отдельной fork ветке test/tabular-xlsx-close-errors-20261115, HEAD0594980b4de75fac38c44b7023624c79688d566d, parent exacta2ba. Ровно4 testfiles/18insertions/4deletions. gofmt/diffcheck только; нового Go/test/SDK/runtime/native не было. Branch remote independently ls-remote verified. Исходный a2ba и immutable B25 artifacts/receipts не ретегированы. Полный CI PASS не заявляю; новый testWIP ещё не тестирован. Draft2627 остаётся Draft.

Собственные Go/Chrome/стенды после завершения выключены, новые окна не брались. Свежий reboot checkpoint будет отдельно к17:45, сейчас не объявлен. Попытка Herdr сообщения cc2 получила PermissionDenied: elevated server; его не перезапускал, чужие процессы не трогал. Этот файл — fallback handoff.

Очередь после выпуска: B22 исправление ACL500/403+ordinaryXLSX, сохранённый test-only PG fixture correction и незавершённый native XLSX gate; T8 многолистная книга не начата. Не включены B25.

Generated-with: Codex
