این سند، مشخصات کامل معماری و پایپ‌لاین اجرایی پروژه استخراج گراف دانش و ذخیره‌سازی داده‌های غیرساختاریافته است. این مستند به عنوان **System Architecture Specification** برای پیاده‌سازی خودکار (Vibe Coding) تدوین شده است.

---

### **System Architecture Specification**

#### **۱. مرور کلی سیستم (System Overview)**

سیستم یک خط لوله پردازش غیرهمزمان (Event-Driven / Async Pipeline) به زبان **Go** است که داده‌های متنی را از پوشه ورودی دریافت، متن خام و متاداده‌ها را مستقیماً در **PocketBase** ذخیره، پردازش‌های سنگین AI را صف‌بندی، و گراف دانش (Entities & Relations) را استخراج و تطبیق (Deduplication) می‌دهد. سیستم به‌صورت ۱۰۰٪ آفلاین و متکی بر مدل زبانی محلی روی **Ollama (CPU-only)** اجرا می‌شود.

---

#### **۲. ساختار پوشه‌ها و صف‌های کاری (Directory Structure)**

```text
/workspace
  ├── /inbox               # پوشه ورودی فایل‌ها (All formats)
  ├── /processed           # بایگانی فایل‌های موفق
  ├── /failed              # بایگانی فایل‌های خراب/پردازش نشده
  ├── /media_pending       # بایگانی فایل‌های مالتی‌مدیا جهت فازهای بعدی
  ├── /queue
  │     ├── /chunks        # صف چانک‌های متنی منتظر پردازش AI
  │     └── /json          # صف خروجی‌های JSON دریافت شده از LLM
  └── /logs                # لاگ سیستم

```

---

#### **۳. اسکیما دیتابیس PocketBase (Database Schema)**

* **جدول `articles` (اسناد خام):**
* `id` (Text, Primary Key)
* `title` (Text)
* `raw_content` (Text, Long)
* `file_name` (Text)
* `file_path` (Text)
* `file_type` (Text)
* `status` (Select: `pending`, `processing`, `completed`, `failed`)
* `created_at` (DateTime)


* **جدول `entities` (موجودیت‌ها):**
* `id` (Text, Primary Key)
* `name` (Text)
* `clean_name` (Text, Indexed) — *نام نرمال‌شده برای جستجو*
* `type` (Text) — *شخص، سازمان، مکان، رویداد و...*
* `attributes` (JSON) — *فیلدهای پویا و اختصاصی هر موجودیت*


* **جدول `aliases` (مترادفات و نام‌های مستعار):**
* `id` (Text, Primary Key)
* `entity_id` (Relation -> entities)
* `alias_name` (Text, Indexed)


* **جدول `relations` (روابط گراف دانش):**
* `id` (Text, Primary Key)
* `source_entity_id` (Relation -> entities)
* `target_entity_id` (Relation -> entities)
* `relation_type` (Text)
* `article_id` (Relation -> articles)



---

#### **۴. معماری پردازش غیرهمزمان (Daemon Architecture)**

سیستم از ۳ پردازش/ورکر (Worker) کاملاً مستقل و مجزا به زبان Go تشکیل شده است:

```text
┌─────────────────────────────────────────────────────────────────────────┐
│ WORKER 1: Ingestion & Chunking Daemon                                   │
│ [inbox] ──> Read File ──> Save Raw to PB ──> Chunk Text ──> [/queue/chunks] │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ WORKER 2: AI Extraction Daemon                                          │
│ [/queue/chunks] ──> Ollama API ──> Retry Logic ──> [/queue/json]        │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ WORKER 3: Graph Ingestion & Deduplication Daemon                        │
│ [/queue/json] ──> Clean/Normalize ──> Exact & Fuzzy Match ──> PB Graph │
└─────────────────────────────────────────────────────────────────────────┘

```

---

#### **۵. جزئیات فنی اجرای ورکرها (Technical Implementation)**

##### **ورکر ۱: Ingestion & Chunking**

1. اسکن پوشه `/inbox`.
2. اگر فرمت مالتی‌مدیا (صوت/تصویر) بود $\rightarrow$ انتقال مستقیم به `/media_pending`.
3. استخراج متن خام متون (`.docx`, `.pdf`, `.txt`, `.html`, `.csv`, `.md`).
4. درج رکورد در جدول `articles` با متاداده‌ها و وضعیت `pending`.
5. **منطق چانکینگ:**
* بررسی وجود تیترهای ساختاریافته (Heading/Title).
* اگر تیتر موجود بود: تفکیک بر اساس تیترها (هر تیتر = ۱ چانک).
* اگر بدون تیتر بود: چانکینگ فرضی ۲۰۰۰ کاراکتری با **هم‌پوشانی (Overlap) ۳۰۰ کاراکتری**.


6. ذخیره هر چانک به عنوان یک فایل متنی مجزا با متاداده `article_id` در پوشه `/queue/chunks`.
7. انتقال فایل اصلی از `/inbox` به `/processed`.

##### **ورکر ۲: AI Extraction**

1. پایش پیوسته پوشه `/queue/chunks`.
2. ارسال فایل چانک به Ollama REST API با پرامپت استخراج JSON ساختاریافته:
* **خروجی الزامی LLM:**
```json
{
  "entities": [{"name": "", "type": "", "attributes": {}}],
  "relations": [{"source": "", "target": "", "relation_type": ""}]
}

```




3. **مدیریت خطا (Error Handling):**
* در صورت فشل شدن یا خروجی JSON نامعتبر: حداکثر **۳ بار Retry**.
* در صورت شکست پس از ۳ بار تلاش: انتقال فایل چانک به `/failed` و آپدیت وضعیت سند در `articles` به `failed`.


4. ذخیره خروجی معتبر JSON در پوشه `/queue/json` و حذف چانک از `/queue/chunks`.

##### **ورکر ۳: Graph Ingestion & Deduplication**

1. پایش پیوسته پوشه `/queue/json`.
2. خواندن هر فایل JSON و اجرای **الگوریتم تطبیق در Go**:
* **Step A (Normalization):** تبدیل نام‌ها به حروف یکسان (تبدیل «ي/ك» به «ی/ک»، حذف اعراب و علائم نگارشی) و تولید `clean_name`.
* **Step B (Exact Match):** کوئری به PocketBase روی جداول `entities` و `aliases`. اگر یافت شد $\rightarrow$ استفاده از `entity_id` موجود.
* **Step C (Fuzzy Match):** اگر یافت نشد، اجرای محاسبات ریاضی Levenshtein Distance روی `clean_name` موجودیت‌های هم‌نوع. اگر درصد شباهت بالا بود $\rightarrow$ ثبت نام جدید به عنوان `alias` برای همان موجودیت.
* **Step D (Insert):** اگر کاملاً جدید بود $\rightarrow$ ایجاد رکورد جدید در `entities`.


3. ثبت تمام اتصالات شبکه در جدول `relations` همراه با `article_id`.
4. آپدیت وضعیت سند در `articles` به `completed` و حذف فایل از `/queue/json`.