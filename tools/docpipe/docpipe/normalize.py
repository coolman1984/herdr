"""Type normalisation: numbers (Arabic-Indic digits, separators, parentheses, percent), money as Decimal,
dates (ISO, d/m/y, Excel serials), currencies. Nothing is guessed silently: every result says how it was
read, and anything ambiguous keeps its raw text with confidence < 1."""
import datetime as _dt
import re
from decimal import Decimal, InvalidOperation

_DIGITS = str.maketrans("٠١٢٣٤٥٦٧٨٩۰۱۲۳۴۵۶۷۸۹", "01234567890123456789")
_CURRENCY = [
    (re.compile(r"(?i)\b(EGP|LE|L\.E\.?)\b|ج\.?م\.?|جنيه"), "EGP"),
    (re.compile(r"(?i)\bSAR\b|ر\.?س\.?|ريال"), "SAR"),
    (re.compile(r"(?i)\bAED\b|د\.?إ\.?|درهم"), "AED"),
    (re.compile(r"(?i)\bUSD\b|US\$|\$"), "USD"),
    (re.compile(r"(?i)\bEUR\b|€"), "EUR"),
    (re.compile(r"(?i)\bGBP\b|£"), "GBP"),
]
_CUR_TOKENS = re.compile(r"(?i)(?<![a-z])(EGP|LE|L\.E\.?|SAR|AED|USD|US\$|EUR|GBP)(?![a-z])|[$€£]|ج\.?\s?م\.?|جنيه(ا|ات)?|ر\.?\s?س\.?|ريال(ا|ات)?|د\.?\s?إ\.?|درهم(ا)?")
_NUM = re.compile(r"^[+-]?(\d{1,3}([,\s٬]\d{3})+|\d+)?([.٫]\d+)?$")


def to_ascii_digits(s):
    return s.translate(_DIGITS)


def currency_of(text):
    for rx, code in _CURRENCY:
        if rx.search(text or ""):
            return code
    return None


def parse_number(raw):
    """Return (Decimal, info) or (None, reason). Accepts 1,234.50  (1,234)  -12%  ١٢٣٫٥  1 234."""
    if raw is None:
        return None, "empty"
    s = to_ascii_digits(str(raw)).strip()
    if not s:
        return None, "empty"
    info = []
    cur = currency_of(s)
    if cur:
        info.append("currency " + cur)
    # Only currency tokens are removed; any other word means this is text that contains a number
    # ("عميل 17", "Invoice 1001"), not a number.
    s = _CUR_TOKENS.sub(" ", s).strip()
    neg = False
    if s.startswith("(") and s.endswith(")"):
        neg, s = True, s[1:-1].strip()
        info.append("parentheses negative")
    pct = s.endswith("%")
    if pct:
        s = s[:-1].strip()
    s = s.replace("٫", ".").replace("٬", ",")
    if s.count(",") and s.count(".") == 0 and re.fullmatch(r"\d+,\d{1,2}", s):
        s = s.replace(",", ".")  # 12,5 -> decimal comma
        info.append("decimal comma")
    if not _NUM.match(s.replace("-", "", 1).replace("+", "", 1)) and not _NUM.match(s):
        return None, "not a number"
    s = s.replace(",", "").replace(" ", "").replace(" ", "")
    try:
        v = Decimal(s)
    except InvalidOperation:
        return None, "not a number"
    if neg:
        v = -v
    if pct:
        v = v / Decimal(100)
        info.append("percent")
    return v, ", ".join(info)


_EXCEL_EPOCH = _dt.datetime(1899, 12, 30)


def excel_serial_to_date(n, date1904=False):
    """Excel serial -> datetime. The 1900 system copies Lotus' fake 1900-02-29 (serial 60): serials 1-59
    count from 1899-12-31, 60 is refused, 61+ from 1899-12-30. The 1904 system counts from 1904-01-01."""
    n = Decimal(str(n))
    if date1904:
        base = _dt.datetime(1904, 1, 1)
    elif n < 60:
        base = _dt.datetime(1899, 12, 31)
    elif n < 61:
        if n >= 60:
            raise ValueError("serial 60 is 1900-02-29, a date that does not exist (Excel's 1900 leap-year bug)")
        base = _dt.datetime(1899, 12, 31)
    else:
        base = _EXCEL_EPOCH
    days = int(n)
    secs = round((n - days) * 86400)
    return base + _dt.timedelta(days=days, seconds=secs)


def parse_date(raw, dayfirst=True):
    """Return (iso-string, confidence, how) or (None, 0, reason). d/m/y is the default (Egypt, Gulf);
    a date that reads both ways (03/04/2026) gets confidence 0.6."""
    if raw is None:
        return None, 0, "empty"
    s = to_ascii_digits(str(raw)).strip()
    m = re.fullmatch(r"(\d{4})-(\d{1,2})-(\d{1,2})(?:[ T](\d{1,2}):(\d{2})(?::(\d{2}))?)?", s)
    if m:
        y, mo, d = int(m[1]), int(m[2]), int(m[3])
        try:
            return _dt.date(y, mo, d).isoformat(), 1.0, "ISO"
        except ValueError:
            return None, 0, "invalid ISO date"
    m = re.fullmatch(r"(\d{1,2})[/.\-](\d{1,2})[/.\-](\d{2,4})", s)
    if m:
        a, b, y = int(m[1]), int(m[2]), int(m[3])
        if y < 100:
            y += 2000
        d, mo = (a, b) if dayfirst else (b, a)
        conf = 1.0 if (a > 12 or b > 12) else 0.6
        if a > 12 and not dayfirst:
            d, mo = a, b
        if b > 12 and dayfirst:
            d, mo = b, a
        try:
            return _dt.date(y, mo, d).isoformat(), conf, "d/m/y" if dayfirst else "m/d/y"
        except ValueError:
            return None, 0, "invalid date"
    return None, 0, "not a date"


def money(v, places=2):
    """Round half-up to places (accounting); never float."""
    q = Decimal(1).scaleb(-places)
    return Decimal(v).quantize(q, rounding="ROUND_HALF_UP")
