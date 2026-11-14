"""Import-only Stage2 callback. No login, files, cookies, process or browser API.

The cc2b-reviewed owner coordinator supplies an already authenticated dedicated
SyncPlaywright page/context. This module never imports the old login harness.
No import-time I/O, environment access, tracing, listeners or browser actions.
"""
from collections.abc import Mapping
from types import MappingProxyType
from urllib.parse import parse_qsl, quote, unquote, urlsplit
from uuid import UUID
import time


class CaptureStopped(Exception):
    """Only this closed message may cross the owner callback boundary."""


ORIGINS = frozenset(('http://127.0.0.1:19548', 'http://127.0.0.1:18548'))
# Canonical frozen metadata names; destination shape does not grant ACL.
_METADATA_SUBSYSTEMS = frozenset(('Документы', 'Касса', 'КоллЦентр', 'МРМ', 'Мессенджер', 'ОТК', 'Обработки', 'Производство', 'Регистры', 'Склад', 'Справочники'))


def _on_origin(url: str, origin: str) -> bool:
    parsed = urlsplit(url)
    return (parsed.scheme+'://'+parsed.netloc == origin
            and parsed.username is None and parsed.password is None)


def _native_entry_url(url: str, origin: str) -> bool:
    # Native LoginSubmit -> index/hiddenHomeRedirect -> tabs app shell.
    # Destination shape is not an authorization token or an RLS bypass.
    try:
        if origin not in ORIGINS or not isinstance(url, str):
            return False
        if any(c.isspace() or ord(c) < 32 or ord(c) == 127 for c in url):
            return False
        if not _on_origin(url, origin) or '#' in url:
            return False
        parsed = urlsplit(url)
        if parsed.path not in ('/ui', '/ui/', '/ui/app'):
            return False
        if not parsed.query:
            return '?' not in url and parsed.path in ('/ui', '/ui/')
        raw_pairs = parsed.query.split('&')
        if any('=' not in pair or pair.split('=', 1)[0] not in ('home', 'subsystem')
               for pair in raw_pairs):
            return False
        pairs = parse_qsl(parsed.query, keep_blank_values=True, strict_parsing=True,
                          max_num_fields=2, separator='&')
        if len(pairs) != len({key for key, _ in pairs}):
            return False
        query = dict(pairs)
        if 'subsystem' in query and query['subsystem'] not in _METADATA_SUBSYSTEMS:
            return False
        if parsed.path == '/ui/app':
            return (query.get('home') == '1'
                    and 'home=1' in raw_pairs
                    and set(query).issubset({'home', 'subsystem'}))
        return set(query) == {'subsystem'}
    except (ValueError, UnicodeError, TypeError):
        return False


def capture_authenticated(*, page, context, origin, target_binding) -> None:
    """Synchronous ABI; None only. Context/browser cleanup stays with owner.

    No credential/cookie/storage_state access. No response body, post_data,
    screenshots, HAR, trace, source/title values, raw exceptions or output.
    Backend flags collector is the authoritative server evidence. Client
    observations here are assertions, not a data-export channel.
    """
    started = time.monotonic()
    state = {"inflight": set(), "finished": 0, "last": started,
             "posts": 0, "failure": False, "page_errors": 0}
    frame = None
    installed = False
    listeners = []
    try:
        if origin not in ORIGINS or page.context is not context or len(context.pages) != 1:
            raise CaptureStopped("CAPTURE_STOPPED")
        if not isinstance(target_binding, MappingProxyType) or not isinstance(target_binding, Mapping):
            raise CaptureStopped("CAPTURE_STOPPED")
        if set(target_binding) != {"entity", "id"} or target_binding["entity"] != "А_Звонок":
            raise CaptureStopped("CAPTURE_STOPPED")
        ident = target_binding["id"]
        if not isinstance(ident, str) or str(UUID(ident)) != ident:
            raise CaptureStopped("CAPTURE_STOPPED")
        if not _native_entry_url(page.url, origin):
            raise CaptureStopped("CAPTURE_STOPPED")
        entity = target_binding["entity"]
        event_path = "/ui/document/" + entity + "/form-event"

        def is_event(request):
            # These transient URLs never enter an output/receipt/exception.
            return request.method == "POST" and unquote(urlsplit(request.url).path) == event_path

        def requested(request):
            try:
                if is_event(request):
                    state["posts"] += 1
                    state["inflight"].add(request)
                    state["last"] = time.monotonic()
                    # Read exactly one non-secret negotiation header, never
                    # request.headers/all_headers/cookie/authorization.
                    if request.header_value("X-OneBase-UI-Commands") != "1":
                        state["failure"] = True
            except Exception:
                state["failure"] = True

        def finished(request):
            try:
                if request in state["inflight"]:
                    state["inflight"].remove(request)
                    state["finished"] += 1
                    state["last"] = time.monotonic()
            except Exception:
                state["failure"] = True

        def failed(request):
            try:
                if request in state["inflight"]:
                    state["inflight"].remove(request)
                    state["failure"] = True
            except Exception:
                state["failure"] = True

        def page_error(_ignored):
            # Never inspect/str/repr the exception (could contain private data).
            state["page_errors"] += 1

        for name, callback in (("request", requested), ("requestfinished", finished),
                               ("requestfailed", failed), ("pageerror", page_error)):
            page.on(name, callback)
            listeners.append((name, callback))

        # Exact native entry used by the original own header harness. No
        # manual OnOpen/AfterOpen POST, FormData construction or fetch hook.
        page.goto(origin + "/ui/document/" + quote(entity) + "/" + ident,
                  wait_until="domcontentloaded", timeout=15000)
        deadline = min(started + 60.0, time.monotonic() + 15.0)
        while time.monotonic() < deadline:
            if state["failure"] or state["page_errors"]:
                raise CaptureStopped("CAPTURE_STOPPED")
            candidates = []
            for candidate in page.frames:
                try:
                    if candidate.locator('#main-form [data-ob-audio]').count() or candidate.locator('[data-ob-audio]').count():
                        if candidate.locator('[data-ob-audio][data-audio-name="ПлеерЗаписи"]').count() == 1:
                            candidates.append(candidate)
                except Exception:
                    state["failure"] = True
            if len(candidates) == 1:
                frame = candidates[0]
                button = frame.locator('[data-audio-name="ПлеерЗаписи"] [data-ob-audio-play]')
                if (state["finished"] > 0 and not state["inflight"]
                        and time.monotonic() - state["last"] >= 0.75
                        and button.count() == 1 and button.is_enabled()):
                    break
            page.wait_for_timeout(25)
        else:
            raise CaptureStopped("CAPTURE_STOPPED")

        if frame is None or frame.evaluate('window._obFormDirty') is not False:
            raise CaptureStopped("CAPTURE_STOPPED")
        # Readonly trusted native input/change counter, installed AFTER login.
        # No DOM values/names/types, no dirty setters or product monkeypatch.
        frame.evaluate('''() => {
          if (window.__s2ReadOnlyCounter) throw new Error('stage2_counter_exists');
          const counter={count:0};
          const observe=e=>{if(e.isTrusted)counter.count++;};
          document.addEventListener('input',observe,true);
          document.addEventListener('change',observe,true);
          window.__s2ReadOnlyCounter={counter,observe};
        }''')
        installed = True
        sent = state["posts"]
        frame.locator('[data-audio-name="ПлеерЗаписи"] [data-ob-audio-play]').click(timeout=7000)
        deadline = min(started + 60.0, time.monotonic() + 15.0)
        while time.monotonic() < deadline:
            if state["failure"] or state["page_errors"]:
                raise CaptureStopped("CAPTURE_STOPPED")
            if state["posts"] > sent and not state["inflight"]:
                frame.wait_for_timeout(25)
                dirty = frame.evaluate('window._obFormDirty')
                changes = frame.evaluate('window.__s2ReadOnlyCounter.counter.count')
                if type(dirty) is not bool or changes != 0:
                    raise CaptureStopped("CAPTURE_STOPPED")
                # Both true (actual RED) and false (not reproduced) are facts.
                # Do not force either, retry, Play again, Close, Save or edit.
                return None
            page.wait_for_timeout(25)
        raise CaptureStopped("CAPTURE_STOPPED")
    except Exception:
        # No underlying message, object/URL, traceback or chained exception.
        raise CaptureStopped("CAPTURE_STOPPED") from None
    finally:
        if installed and frame is not None:
            try:
                frame.evaluate('''() => {
                  const h=window.__s2ReadOnlyCounter;
                  if(h){document.removeEventListener('input',h.observe,true);
                        document.removeEventListener('change',h.observe,true);
                        delete window.__s2ReadOnlyCounter;}
                }''')
            except Exception:
                pass
        for name, callback in listeners:
            try:
                page.remove_listener(name, callback)
            except Exception:
                pass
        # Owner coordinator finally closes context; future owner runner
        # finally closes browser and its own units/tunnel. Not owned here.


if __name__ == "__main__":
    raise SystemExit(2)  # import-only: no CLI execution path
