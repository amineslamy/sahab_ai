ترجمه کامل، دقیق و فنی فایل `AGENT_GUIDE.md` آماده شد. می‌توانید این محتوا را در فایل مربوطه در پروژه خود قرار دهید:

---

# 🧠 ساخت عامل‌های هوش مصنوعی با GraphMem

> **راهنمای مرجع و جامع برای ساخت عامل‌های هوش مصنوعی عملیاتی با حافظه شبیه به انسان**
> 

پروژه GraphMem لایه حافظه‌ای را ارائه می‌دهد که مدل‌های بزرگ زبانی (LLM) بدون حالت (Stateless) را به عامل‌های هوشمندی تبدیل می‌کند که قادر به یادگیری، به‌خاطرسپاری و تکامل در طول زمان هستند.

---

## Table of Contents

1. [Why GraphMem for Agents?](#why-graphmem-for-agents)
2. [Quick Start](#quick-start)
3. [Core Concepts](#core-concepts)
4. [Agent Patterns](#agent-patterns)
5. [Production Architecture](#production-architecture)
6. [Advanced Features](#advanced-features)
7. [Best Practices](#best-practices)
8. [Complete Examples](#complete-examples)

## فهرست مطالب

۱. [چرا GraphMem برای عامل‌ها؟](https://www.google.com/search?q=%23%DA%86%D8%B1%D8%A7-graphmem-%D8%A8%D8%B1%D8%A7%DB%8C-%D8%B9%D8%A7%D9%85%D9%84%E2%80%8C%D9%87%D8%A7)

۲. [شروع سریع](https://www.google.com/search?q=%23%D8%B4%D8%B1%D9%88%D8%B9-%D8%B3%D8%B1%DB%8C%D8%B9)

۳. [مفاهیم پایه](https://www.google.com/search?q=%23%D9%85%D9%81%D8%A7%D9%87%DB%8C%D9%85-%D9%BE%D8%A7%DB%8C%D9%87)

۴. [الگوهای طراحی عامل‌ها](https://www.google.com/search?q=%23%D8%A7%D9%84%DA%AF%D9%88%D9%87%D8%A7%DB%8C-%D8%B7%D8%B1%D8%A7%D8%AD%DB%8C-%D8%B9%D8%A7%D9%85%D9%84%E2%80%8C%D9%87%D8%A7)

۵. [معماری محیط عملیاتی](https://www.google.com/search?q=%23%D9%85%D8%B9%D9%85%D8%A7%D8%B1%DB%8C-%D9%85%D8%AD%DB%8C%D8%B7-%D8%B9%D9%85%D9%84%DB%8C%D8%A7%D8%AA%DB%8C)

۶. [ویژگی‌های پیشرفته](https://www.google.com/search?q=%23%D9%88%DB%8C%DA%98%DA%AF%DB%8C%E2%80%8C%D9%87%D8%A7%DB%8C-%D9%BE%DB%8C%D8%B4%D8%B1%D9%81%D8%AA%D9%87)

۷. [بهترین تجربیات (Best Practices)](https://www.google.com/search?q=%23%D8%A8%D9%87%D8%AA%D8%B1%DB%8C%D9%86-%D8%AA%D8%AC%D8%B1%D8%A8%DB%8C%D8%A7%D8%AA-best-practices)

۸. [نمونه‌های کامل کد](https://www.google.com/search?q=%23%D9%86%D9%85%D9%88%D9%86%D9%87%E2%80%8C%D9%87%D8%A7%DB%8C-%DA%A9%D8%A7%D9%85%D9%84-%DA%A9%D8%AF)

---

## چرا GraphMem برای عامل‌ها؟

### مشکل حافظه‌های فعلی در عامل‌ها

| رویکرد | مشکل |
| --- | --- |
| **تزریق مستقیم به پرامپت (Context stuffing)** | محدودیت توکن، هزینه‌بر، عدم یادگیری

 |
| **پایگاه داده برداری ساده (Vector DB)** | عدم درک روابط، عدم درک زمان

 |
| **پایگاه داده کلید-مقدار (Key-value)** | عدم درک معنایی (Semantic)

 |
| **روش RAG سنتی** | بازیابی تکه‌های متن بدو درک دانش

 |

### GraphMem چگونه این مشکل را حل می‌کند؟

```text
┌─────────────────────────────────────────────────────────────────┐
│                   سیستم حافظه GRAPHMEM                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  📥 دریافت (INGEST)   🧠 درک (UNDERSTAND)     📤 بازیابی (RETRIEVE)│
│  ─────────────────   ───────────────────     ───────────────────│
│  • متون ساده         • استخراج موجودیت‌ها      • جستجوی معنایی    │
│  • اسناد             • نگاشت روابط           • پیمایش گراف      │
│  • گفتگوها           • حل نام‌های مستعار      • پرس‌وجوی انجمنی   │
│                      • ردیابی زمانی          • استدلال چندمرحله‌ای│
│                                                                  │
│  🔄 تکامل (EVOLVE)   ⚡ بهینه‌سازی (OPTIMIZE)  🔒 جداسازی (ISOLATE)│
│  ─────────────────   ──────────────────────  ───────────────────│
│  • تثبیت دانش        • رتبه‌دهی PageRank      • چندمستأجره       │
│  • زوال داده قدیمی   • وزن‌دهی اهمیت          • ایزوله‌سازی کاربر │
│  • رفع تناقضات       • کش‌سازی (Redis)        • مدیریت محدوده جلسه│
│                                                                  │
└─────────────────────────────────────────────────────────────────┘

```

---

## شروع سریع

### نصب

```bash
# هسته اصلی (صرفاً در حافظه - برای تست)
pip install agentic-graph-mem

# همراه با ماندگاری داده (پیشنهادشده برای محیط عملیاتی)
pip install "agentic-graph-mem[libsql]"

```

### اولین عامل مجهز به حافظه شما

> ⚠️ **هشدار مهم: ماندگاری داده‌ها را فعال کنید!**
> 
> بدون تنظیم `turso_db_path`، عامل شما با هر بار راه اندازی مجدد تمام حافظه خود را فراموش می‌کند!
> 
> 

```python
from graphmem import GraphMem, MemoryConfig

# پیکربندی اولیه با سرویس‌دهنده LLM
config = MemoryConfig(
    # LLM برای استخراج و پرس‌وجو
    llm_provider="openai",
    llm_api_key="sk-...",
    llm_model="gpt-4o-mini",
    
    # بردارسازی (Embeddings) برای جستجوی معنایی
    embedding_provider="openai",
    embedding_api_key="sk-...",
    embedding_model="text-embedding-3-small",
    
    # ✅ حیاتی: ذخیره‌سازی ماندگار!
    turso_db_path="agent_memory.db",
    
    # فعال‌سازی خودتکاملی
    evolution_enabled=True,
    auto_evolve=True,  # تکامل پس از هر بار دریافت داده
)

# ساخت نمونه حافظه با memory_id و user_id یکسان
memory = GraphMem(config, memory_id="my_agent", user_id="default")

# اکنون عامل شما می‌تواند یاد بگیرد و به خاطر بسپارد!
class SimpleAgent:
    def __init__(self, memory: GraphMem):
        self.memory = memory
    
    def learn(self, information: str):
        """یادگیری اطلاعات جدید توسط عامل"""
        self.memory.ingest(information)
    
    def ask(self, question: str) -> str:
        """پاسخ‌گویی عامل بر اساس حافظه"""
        response = self.memory.query(question)
        return response.answer
    
    def reflect(self):
        """تثبیت و بهبود حافظه توسط عامل"""
        self.memory.evolve()

# استفاده از عامل
agent = SimpleAgent(memory)
agent.learn("تسلا در سال ۲۰۰۳ تأسیس شد. ایلان ماسک در سال ۲۰۰۸ مدیرعامل شد.")
agent.learn("اسپیس‌ایکس در سال ۲۰۰۲ توسط ایلان ماسک تأسیس شد.")

print(agent.ask("ایلان ماسک چه شرکت‌هایی را تأسیس کرده است؟"))
# ← "ایلان ماسک اسپیس‌ایکس را در سال ۲۰۰۲ تأسیس کرد و در سال ۲۰۰۸ مدیرعامل تسلا شد."

agent.reflect()  # تثبیت دانش در مورد ایلان ماسک

```

---

## مفاهیم پایه

### ۱. سه رکن اصلی: دریافت (Ingest) ← پرس‌وجو (Query) ← تکامل (Evolve)

```python
# دریافت: تزریق اطلاعات به حافظه
memory.ingest("""
    انتروپیک در سال ۲۰۲۱ توسط داریو آمودی و دانیلا آمودی تأسیس شد.
    آن‌ها پیش از این در OpenAI کار می‌کردند. انتروپیک مدل کلود (Claude) را خلق کرد.
""")

# پرس‌وجو: پرسیدن سوالات
response = memory.query("چه کسی انتروپیک را تأسیس کرد؟")
print(response.answer)      # "داریو آمودی و دانیلا آمودی انتروپیک را در سال ۲۰۲۱ تأسیس کردند"
print(response.confidence)  # 0.95
print(response.context)     # متن کاملی که برای پاسخ استفاده شده است

# تکامل: بهبود حافظه در طول زمان
events = memory.evolve()
for event in events:
    print(f"{event.evolution_type}: {event.description}")
    # تثبیت: ادغام ۳ بار اشاره به "انتروپیک" در ۱ موجودیت واحد
    # زوال: آرشیو کردن ۲ رابطه قدیمی و منقضی‌شده

```

### ۲. ساختار گراف دانش

پروژه GraphMem به صورت خودکار یک گراف دانش از متن شما می‌سازد:

```text
                    ┌─────────────┐
                    │  Anthropic  │
                    │  (شرکت)     │
                    └──────┬──────┘
                           │
           ┌───────────────┼───────────────┐
           │               │               │
           ▼               ▼               ▼
    ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
    │Dario Amodei │ │Daniela      │ │   Claude    │
    │   (شخص)     │ │Amodei       │ │   (محصول)   │
    │             │ │(شخص)        │ │             │
    └─────────────┘ └─────────────┘ └─────────────┘
           │               │
           └───────┬───────┘
                   │
                   ▼
            ┌─────────────┐
            │   OpenAI    │
            │   (شرکت)    │
            └─────────────┘

روابط:
• Dario Amodei --[تأسیس کرد]--> Anthropic (از ۲۰۲۱ تا کنون)
• Dario Amodei --[کار می‌کرد در]--> OpenAI (گذشته)
• Anthropic --[خلق کرد]--> Claude

```

### ۳. استخراج همه‌جانبه

سیستم GraphMem **تمام** جزئیات متن را استخراج می‌کند:

```python
memory.ingest("""
    در سه‌ماهه سوم ۲۰۲۴، شرکت انویدیا (NVDA) درآمد ۳۵.۱ میلیارد دلاری با رشد ۹۴ درصدی نسبت به سال قبل گزارش داد.
    جنسن هوانگ، مدیرعامل شرکت، اعلام کرد ارسال Blackwell B200 در اوایل سال ۲۰۲۵ آغاز می‌شود.
    این شرکت ۲۶,۰۰۰ کارمند در سراسر جهان دارد.
""")

# خروجی استخراج‌شده توسط GraphMem:
# موجودیت‌ها (ENTITIES):
# - انویدیا (شرکت) [نام‌های مستعار: NVDA, Nvidia Corporation]
# - جنسن هوانگ (شخص) [نام‌های مستعار: J. Huang]
# - ۳۵.۱ میلیارد دلار (مبلغ)
# - ۹۴٪ (درصد)
# - سه‌ماهه سوم ۲۰۲۴ (تاریخ)
# - Blackwell B200 (محصول)
# - ۲۶,۰۰۰ (عدد)
# - اوایل ۲۰۲۵ (تاریخ)

# روابط (RELATIONSHIPS):
# - جنسن هوانگ --[مدیرعامل است در]--> انویدیا
# - انویدیا --[گزارش درآمد داد]--> ۳۵.۱ میلیارد دلار [اعتبار: سه‌ماهه سوم ۲۰۲۴]
# - انویدیا --[دستیابی به رشد]--> ۹۴٪
# - انویدیا --[دارای کارمندان]--> ۲۶,۰۰۰
# - Blackwell B200 --[ارسال می‌شود در]--> اوایل ۲۰۲۵

```

### ۴. حفظ تکه‌های متن مبدأ

هر موجودیت **متن اصلی مبدأ** خود را برای ارائه پاسخ‌های دقیق حفظ می‌کند:

```python
# هنگامی که پرس‌وجو می‌کنید، GraphMem موارد زیر را ارائه می‌دهد:
response = memory.query("درآمد انویدیا چقدر است؟")

# مدل LLM موارد زیر را مشاهده می‌کند:
# ۱. متن اصلی مبدأ (بافت کامل)
# ۲. موجودیت‌های استخراج‌شده (ساختاریافته)
# ۳. روابط (اتصالات)
# ۴. خلاصه گروه‌ها و جوامع دانش (درک سطح بالا)

```

---

## الگوهای طراحی عامل‌ها

### الگوی اول: عامل گفتگو محور با حافظه ماندگار

```python
from graphmem import GraphMem, MemoryConfig, MemoryImportance

class ConversationalAgent:
    """عاملی که گفتگوها را در جلسات مختلف به خاطر می‌سپارد."""
    
    def __init__(self, user_id: str):
        self.config = MemoryConfig(
            llm_provider="openai",
            llm_api_key="sk-...",
            llm_model="gpt-4o-mini",
            embedding_provider="openai",
            embedding_api_key="sk-...",
            embedding_model="text-embedding-3-small",
            
            # ذخیره‌سازی در SQLite
            turso_db_path=f"memories/{user_id}.db",
            
            # جداسازی کاربران
            user_id=user_id,
            
            # تنظیمات تکامل
            evolution_enabled=True,
            decay_enabled=True,
            decay_half_life_days=30,  # فراموشی اطلاعات استفاده‌نشده پس از حدود ۳۰ روز
        )
        self.memory = GraphMem(self.config, memory_id=f"agent_{user_id}", user_id=user_id)
        self.user_id = user_id
    
    def chat(self, user_message: str) -> str:
        """پردازش پیام کاربر و تولید پاسخ."""
        
        # ۱. ذخیره پیام کاربر به عنوان حافظه
        self.memory.ingest(
            f"کاربر گفت: {user_message}",
            metadata={"type": "user_message", "timestamp": "now"},
            importance=MemoryImportance.MEDIUM,
        )
        
        # ۲. پرس‌وجو از حافظه برای یافتن بافت مرتبط
        response = self.memory.query(user_message)
        
        # ۳. تولید پاسخ (استفاده از مدل LLM شما)
        agent_response = self._generate_response(user_message, response.context)
        
        # ۴. ذخیره پاسخ عامل
        self.memory.ingest(
            f"پاسخ عامل: {agent_response}",
            metadata={"type": "agent_response"},
            importance=MemoryImportance.LOW,
        )
        
        return agent_response
    
    def _generate_response(self, query: str, context: str) -> str:
        """تولید پاسخ با استفاده از LLM و بافت حافظه."""
        pass
    
    def end_session(self):
        """تثبیت حافظه در پایان جلسه گفتگو."""
        self.memory.evolve()

# نحوه استفاده
agent = ConversationalAgent(user_id="user_123")
agent.chat("اسم من آلیس است و در گوگل کار می‌کنم.")
agent.chat("من به یادگیری ماشین علاقه‌مندم.")

# در جلسات بعدی...
agent.chat("درباره من چه می‌دانی؟")
# ← "شما آلیس هستید، در گوگل کار می‌کنید و به یادگیری ماشین علاقه دارید."

```

### الگوی دوم: عامل پژوهشگر با یادگیری چندسندی

```python
class ResearchAgent:
    """عاملی که از چندین سند یاد می‌گیرد و به سوالات پیچیده پاسخ می‌دهد."""
    
    def __init__(self):
        self.config = MemoryConfig(
            llm_provider="azure_openai",
            llm_api_key="...",
            llm_api_base="https://your-resource.openai.azure.com/",
            azure_deployment="gpt-4",
            llm_model="gpt-4",
            azure_api_version="2024-02-15-preview",
            
            embedding_provider="azure_openai",
            embedding_api_key="...",
            embedding_api_base="https://your-resource.openai.azure.com/",
            azure_embedding_deployment="text-embedding-ada-002",
            embedding_model="text-embedding-ada-002",
            
            # استفاده از Neo4j برای ذخیره‌سازی گرافی در محیط عملیاتی
            neo4j_uri="neo4j+s://your-instance.databases.neo4j.io",
            neo4j_username="neo4j",
            neo4j_password="...",
            
            # استفاده از Redis برای کش‌سازی
            redis_url="redis://...",
            
            # تکامل جدی برای کارهای پژوهشی
            evolution_enabled=True,
            consolidation_threshold=0.75,  # ادغام شدیدتر
        )
        self.memory = GraphMem(self.config, memory_id="research_agent", user_id="researcher")
    
    def ingest_documents(self, documents: list[dict]):
        """دریافت دسته‌ای و کارآمد چندین سند."""
        result = self.memory.ingest_batch(
            documents,
            max_workers=20,  # پردازش موازی
            aggressive=True,
            show_progress=True,
        )
        print(f"پردازش شد: {result['documents_processed']} سند")
        print(f"استخراج شد: {result['total_entities']} موجودیت")
        print(f"یافت شد: {result['total_relationships']} رابطه")
        
        # اجرای تکامل پس از دریافت دسته‌ای
        self.memory.evolve()
    
    def research(self, question: str) -> dict:
        """پاسخ به سوالات پژوهشی و پیچیده."""
        response = self.memory.query(question)
        
        return {
            "answer": response.answer,
            "confidence": response.confidence,
            "sources": [n.name for n in response.nodes[:5]],
            "related_entities": [n.name for n in response.nodes],
            "context_tokens": len(response.context.split()),
        }
    
    def find_connections(self, entity_a: str, entity_b: str) -> str:
        """یافتن نحوه ارتباط دو موجودیت."""
        query = f"ارتباط یا اتصال بین {entity_a} و {entity_b} چگونه است؟"
        response = self.memory.query(query)
        return response.answer

# نحوه استفاده
agent = ResearchAgent()

# دریافت مقالات پژوهشی
papers = [
    {"id": "paper1", "content": "...مقاله در مورد ترنسفورمرها..."},
    {"id": "paper2", "content": "...مقاله در مورد مکانیسم‌های توجه..."},
    {"id": "paper3", "content": "...مقاله در مورد معماری GPT..."},
]
agent.ingest_documents(papers)

# پرسش سوالات پیچیده
result = agent.research("مکانیسم‌های توجه چگونه به مدل‌های زبانی بزرگ امروزی تبدیل شدند؟")
print(result["answer"])

# یافتن روابط بین موجودیت‌ها
print(agent.find_connections("Transformers", "GPT-4"))

```

### الگوی سوم: عامل پشتیبانی مشتریان با قابلیت رفع تناقضات

```python
class SupportAgent:
    """عاملی برای پاسخ‌گویی به پشتیبانی مشتریان با دانش به‌روز."""
    
    def __init__(self, company_id: str):
        self.config = MemoryConfig(
            llm_provider="openai",
            llm_api_key="sk-...",
            llm_model="gpt-4o",
            embedding_provider="openai",
            embedding_api_key="sk-...",
            embedding_model="text-embedding-3-small",
            
            turso_db_path=f"support/{company_id}.db",
            
            # حیاتی: فعال‌سازی اعتبار زمانی برای تغییر قوانین و سیاست‌ها
            evolution_enabled=True,
            decay_enabled=True,
        )
        self.memory = GraphMem(self.config, memory_id="research_agent", user_id="researcher")
    
    def update_knowledge(self, content: str, importance: str = "HIGH"):
        """به‌روزرسانی پایگاه دانش با اطلاعات جدید."""
        from graphmem import MemoryImportance
        
        importance_map = {
            "CRITICAL": MemoryImportance.CRITICAL,
            "HIGH": MemoryImportance.HIGH,
            "MEDIUM": MemoryImportance.MEDIUM,
            "LOW": MemoryImportance.LOW,
        }
        
        self.memory.ingest(
            content,
            importance=importance_map.get(importance, MemoryImportance.MEDIUM),
        )
        
        # حیاتی: اجرای تکامل برای حل تناقضات با اطلاعات قدیمی
        self.memory.evolve()
    
    def answer_ticket(self, ticket: str) -> dict:
        """پاسخ به تیکت پشتیبانی بر اساس پایگاه دانش."""
        response = self.memory.query(ticket)
        
        return {
            "answer": response.answer,
            "confidence": response.confidence,
            "needs_escalation": response.confidence < 0.6,
        }

# نحوه استفاده
agent = SupportAgent("acme_corp")

# دانش اولیه
agent.update_knowledge("""
    قوانین مرجوعی ما اجازه بازگشت کالا تا ۳۰ روز پس از خرید را می‌دهد.
    استرداد وجه ظرف ۵ تا ۷ روز کاری انجام می‌شود.
""")

# به‌روزرسانی قوانین - GraphMem تناقض را حل خواهد کرد!
agent.update_knowledge("""
    به‌روزرسانی قوانین (۲۰۲۴): قوانین جدید مرجوعی اجازه بازگشت کالا تا ۶۰ روز را می‌دهد.
    استرداد وجه اکنون ظرف ۲ تا ۳ روز کاری انجام می‌شود.
""", importance="CRITICAL")

# پاسخ عامل با اطلاعات جدید
result = agent.answer_ticket("شرایط مرجوعی کالا چگونه است؟")
print(result["answer"])
# ← "قوانین مرجوعی ما اجازه بازگشت کالا تا ۶۰ روز را می‌دهد. استرداد وجه ظرف ۲ تا ۳ روز کاری انجام می‌شود."
# (قانون قدیمی ۳۰ روزه به صورت خودکار جایگزین شد)

```

### الگوی چهارم: عامل چندمستأجره (Multi-Tenant) برای سامانه‌های SaaS

```python
class MultiTenantAgent:
    """عاملی برای ارائه سرویس به چند مشتری با حافظه‌های کاملاً مجزا."""
    
    def __init__(self, base_config: dict):
        self.base_config = base_config
        self.memories = {}  # tenant_id -> GraphMem
    
    def get_memory(self, tenant_id: str) -> GraphMem:
        """دریافت یا ساخت حافظه برای یک مستأجر (Tenant)."""
        if tenant_id not in self.memories:
            config = MemoryConfig(
                **self.base_config,
                
                # حیاتی: ایزوله‌سازی مستأجرین
                user_id=tenant_id,
                memory_id=f"tenant_{tenant_id}",
                
                # اختصاص پایگاه داده مجزا برای هر مستأجر
                turso_db_path=f"tenants/{tenant_id}/memory.db",
            )
            self.memories[tenant_id] = GraphMem(config, memory_id=f"tenant_{tenant_id}", user_id=tenant_id)
        
        return self.memories[tenant_id]
    
    def ingest(self, tenant_id: str, content: str):
        """ذخیره محتوا برای یک مستأجر خاص."""
        memory = self.get_memory(tenant_id)
        memory.ingest(content)
    
    def query(self, tenant_id: str, question: str) -> str:
        """پرس‌وجو از حافظه یک مستأجر خاص."""
        memory = self.get_memory(tenant_id)
        response = memory.query(question)
        return response.answer

# نحوه استفاده
base_config = {
    "llm_provider": "openai",
    "llm_api_key": "sk-...",
    "llm_model": "gpt-4o-mini",
    "embedding_provider": "openai",
    "embedding_api_key": "sk-...",
    "embedding_model": "text-embedding-3-small",
}

agent = MultiTenantAgent(base_config)

# داده‌های هر مستأجر کاملاً ایزوله است
agent.ingest("acme", "قیمت محصول آکمی ۹۹ دلار است.")
agent.ingest("globex", "قیمت محصول گلوبکس ۱۴۹ دلار است.")

print(agent.query("acme", "قیمت محصول چقدر است؟"))   # ← "۹۹ دلار"
print(agent.query("globex", "قیمت محصول چقدر است؟")) # ← "۱۴۹ دلار"

```

---

## معماری محیط عملیاتی

### پشته تکنولوژی پیشنهادی

```text
┌─────────────────────────────────────────────────────────────────┐
│                    استقرار در محیط عملیاتی                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                     لایه برنامه                          │    │
│  │                                                          │    │
│  │   FastAPI / Flask / Django                               │    │
│  │       ↓                                                  │    │
│  │   نمونه GraphMem (به ازای درخواست یا Singleton)          │    │
│  └─────────────────────────────────────────────────────────┘    │
│                            ↓                                     │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    لایه ذخیره‌سازی                       │    │
│  │                                                          │    │
│  │  Neo4j Aura (گراف)   ←→   Redis Cloud (کش)               │    │
│  │         ↑                        ↑                       │    │
│  │         └────────────────────────┘                       │    │
│  │                    ↓                                     │    │
│  │            Turso (پشتیبان/برداری)                         │    │
│  └─────────────────────────────────────────────────────────┘    │
│                            ↓                                     │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                 سرویس‌دهندگان LLM                         │    │
│  │                                                          │    │
│  │  OpenAI  |  Azure OpenAI  |  Anthropic  |  مدل‌های محلی   │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘

```

### نمونه کد یکپارچه‌سازی با FastAPI

```python
from fastapi import FastAPI, Depends, HTTPException
from pydantic import BaseModel
from graphmem import GraphMem, MemoryConfig
from functools import lru_cache
import os

app = FastAPI()

# نمونه یکتا از حافظه (Singleton)
@lru_cache()
def get_memory() -> GraphMem:
    config = MemoryConfig(
        llm_provider="openai",
        llm_api_key=os.getenv("OPENAI_API_KEY"),
        llm_model="gpt-4o-mini",
        embedding_provider="openai",
        embedding_api_key=os.getenv("OPENAI_API_KEY"),
        embedding_model="text-embedding-3-small",
        
        # ذخیره‌سازی محیط عملیاتی
        neo4j_uri=os.getenv("NEO4J_URI"),
        neo4j_username=os.getenv("NEO4J_USER"),
        neo4j_password=os.getenv("NEO4J_PASSWORD"),
        
        # کش‌سازی با Redis
        redis_url=os.getenv("REDIS_URL"),
        
        evolution_enabled=True,
    )
    return GraphMem(config, memory_id="api_agent", user_id="api_user")

class IngestRequest(BaseModel):
    content: str
    metadata: dict = {}

class QueryRequest(BaseModel):
    question: str

class QueryResponse(BaseModel):
    answer: str
    confidence: float
    sources: list[str]

@app.post("/ingest")
async def ingest(request: IngestRequest, memory: GraphMem = Depends(get_memory)):
    result = memory.ingest(request.content, metadata=request.metadata)
    return {"status": "success", "entities": result["entities"]}

@app.post("/query", response_model=QueryResponse)
async def query(request: QueryRequest, memory: GraphMem = Depends(get_memory)):
    response = memory.query(request.question)
    return QueryResponse(
        answer=response.answer,
        confidence=response.confidence,
        sources=[n.name for n in response.nodes[:5]],
    )

@app.post("/evolve")
async def evolve(memory: GraphMem = Depends(get_memory)):
    events = memory.evolve()
    return {"events": len(events)}

@app.get("/stats")
async def stats(memory: GraphMem = Depends(get_memory)):
    return memory.get_stats()

```

### دریافت دسته‌ای با توان پردازشی بالا

```python
import asyncio
from concurrent.futures import ThreadPoolExecutor

class HighThroughputIngestion:
    """دریافت و پردازش کارآمد میلیون‌ها سند."""
    
    def __init__(self, memory: GraphMem):
        self.memory = memory
    
    def ingest_large_dataset(
        self,
        documents: list[dict],
        batch_size: int = 1000,
        max_workers: int = 20,
    ):
        """دریافت دسته‌ای اسناد همراه با ردیابی پیشرفت."""
        total = len(documents)
        processed = 0
        
        for i in range(0, total, batch_size):
            batch = documents[i:i+batch_size]
            
            result = self.memory.ingest_batch(
                batch,
                max_workers=max_workers,
                aggressive=True,
                show_progress=True,
                rebuild_communities=False,  # موکول کردن ساخت جوامع دانش به پایان کار
            )
            
            processed += result["documents_processed"]
            print(f"پیشرفت: {processed}/{total} ({100*processed/total:.1f}%)")
        
        # بازسازی جوامع دانش در انتها
        print("در حال ساخت جوامع دانش...")
        self.memory.evolve()
        
        return {"total_processed": processed}

# نحوه استفاده
ingestion = HighThroughputIngestion(memory)
ingestion.ingest_large_dataset(
    documents=large_dataset,  # میلیون‌ها سند
    batch_size=1000,
    max_workers=20,
)

```

---

## ویژگی‌های پیشرفته

### ۱. پرس‌وجوهای زمانی (مبتنی بر نقطه زمانی)

```python
# دریافت اطلاعات تاریخی
memory.ingest("""
    استیو جابز از سال ۱۹۹۷ تا ۲۰۱۱ مدیرعامل اپل بود.
    تیم کوک در آگوست ۲۰۱۱ مدیرعامل شد و امروز نیز مدیرعامل است.
""")

# پرس‌وجو درباره گذشته
response = memory.query("در سال ۲۰۰۵ چه کسی مدیرعامل اپل بود؟")
print(response.answer)  # ← "استیو جابز از سال ۱۹۹۷ تا ۲۰۱۱ مدیرعامل اپل بود"

# پرس‌وجو درباره زمان حال
response = memory.query("مدیرعامل فعلی اپل کیست؟")
print(response.answer)  # ← "تیم کوک از آگوست ۲۰۱۱ مدیرعامل است"

```

### ۲. بازیابی هوشمند با درک نام‌های مستعار

```python
# GraphMem نام‌های مستعار را خودکار استخراج و استفاده می‌کند
memory.ingest("""
    دکتر الکساندر چن، معروف به "پیشگام کوانتوم"، 
    آزمایشگاه Quantum AI را تأسیس کرد. الکس چن مدرک دکتری خود را از MIT گرفت.
""")

# تمام این پرس‌وجوها کار می‌کنند:
memory.query("دکتر چن چه کاری انجام داد؟")
memory.query("الکساندر چن کیست؟")
memory.query("درباره پیشگام کوانتوم بگو")
memory.query("الکس چن چه چیزی مطالعه کرد؟")
# همه آن‌ها اطلاعات مربوط به همان شخص واحد را بازمی‌گردانند!

```

### ۳. استدلال چندمرحله‌ای (Multi-Hop Reasoning)

```python
# GraphMem روابط را برای پاسخ به سوالات پیچیده پیمایش می‌کند
memory.ingest("اپل توسط استیو جابز تأسیس شد.")
memory.ingest("استیو جابز شرکت NeXT را نیز تأسیس کرد.")
memory.ingest("شرکت NeXT در سال ۱۹۹۷ توسط اپل خریداری شد.")
memory.ingest("تیم کوک قبل از اپل در Compaq کار می‌کرد.")

# سوال چندمرحله‌ای
response = memory.query("ارتباط بین NeXT و تیم کوک چیست؟")
# مسیر پیمایش: NeXT → استیو جابز → اپل → تیم کوک
print(response.answer)
# ← "شرکت NeXT توسط استیو جابز تأسیس و در سال ۱۹۹۷ توسط اپل خریداری شد. 
#    تیم کوک در حال حاضر به عنوان مدیرعامل در اپل کار می‌کند."

```

### ۴. رفع تناقضات از طریق تکامل

```python
# واقعیت اولیه
memory.ingest("این شرکت ۱,۰۰۰ کارمند دارد.")

# واقعیت به‌روزرسانی‌شده
memory.ingest("خبر فوری: این شرکت پس از توسعه اکنون ۲,۵۰۰ کارمند دارد.")

# تکامل به صورت خودکار تناقض را حل می‌کند
memory.evolve()

# پرس‌وجوها اطلاعات جدیدتر را بازمی‌گردانند
response = memory.query("این شرکت چند کارمند دارد؟")
print(response.answer)  # ← "۲,۵۰۰ کارمند (پس از توسعه)"

```

### ۵. حافظه مبتنی بر میزان اهمیت

```python
from graphmem import MemoryImportance

# اطلاعات حیاتی (هرگز زوال نمی‌یابند)
memory.ingest(
    "حساسیت‌های مشتری: بادام زمینی، صدف دریایی",
    importance=MemoryImportance.CRITICAL,
)

# اطلاعات معمولی (ممکن است در طول زمان کم‌رنگ شوند)
memory.ingest(
    "مشتری اشاره کرد که قهوه دوست دارد",
    importance=MemoryImportance.LOW,
)

# پس از تکامل، حافظه‌های کم‌اهمیت و استفاده‌نشده زوال می‌یابند
# حافظه‌های حیاتی همیشه حفظ می‌شوند

```

---

## بهترین تجربیات (Best Practices)

### ۱. ساختاردهی به ورودی‌ها

```python
# ❌ نادرست: ورود متون آشفته و بی‌ساختار
memory.ingest("امروز یه سری اتفاق افتاد فلان و بیسار...")

# ✅ درست: جملات شفاف و مبتنی بر واقعیت
memory.ingest("""
    در ۱۵ ژانویه ۲۰۲۴، شرکت آکمی درآمد سه‌ماهه چهارم را اعلام کرد:
    - درآمد: ۵.۲ میلیارد دلار (۲۳٪ رشد نسبت به سال قبل)
    - سود خالص: ۸۹۰ میلیون دلار
    - مدیرعامل جین اسمیت این رشد را به محصولات هوش مصنوعی نسبت داد
""")

```

### ۲. استفاده از دریافت دسته‌ای برای حجم داده زیاد

```python
# ❌ نادرست: دریافت تک‌تک و کند
for doc in documents:
    memory.ingest(doc["content"])

# ✅ درست: دریافت دسته‌ای و موازی
memory.ingest_batch(
    documents,
    max_workers=20,
    aggressive=True,
)

```

### ۳. اجرای منظم تکامل (Evolve)

```python
# روش اول: تکامل خودکار (ساده)
config = MemoryConfig(..., auto_evolve=True)

# روش دوم: تکامل صریح (کنترل بیشتر)
# پس از جلسات دریافت داده
memory.ingest_batch(docs)
memory.evolve()

# یا طبق زمان‌بندی (مثلاً روزانه)
import schedule
schedule.every().day.at("02:00").do(memory.evolve)

```

### ۴. انتخاب ذخیره‌سازی مناسب

| مورد استفاده | ذخیره‌سازی پیشنهادی |
| --- | --- |
| **توسعه و تست** | Turso (SQLite) - بدون نیاز به تنظیمات پیچیده |
| **محیط عملیاتی تک‌سرور** | Turso + Redis |
| **محیط عملیاتی چندسرور** | Neo4j + Redis |
| **سازمانی و مقیاس بزرگ** | Neo4j Aura + Redis Cloud |

### ۵. مدیریت صحیح خطاها

```python
from graphmem.core.exceptions import IngestionError, QueryError

try:
    memory.ingest(content)
except IngestionError as e:
    logger.error(f"خطا در دریافت داده: {e}")
    # تلاش مجدد یا قرار دادن در صف

try:
    response = memory.query(question)
except QueryError as e:
    logger.error(f"خطا در پرس‌وجو: {e}")
    # بازگشت به پاسخ پیش‌فرض

```

---

## نمونه‌های کامل کد

### نمونه اول: دستیار دانش شخصی

```python
"""
دستیار دانش شخصی که از یادداشت‌ها، 
مقالات و گفتگوهای شما یاد می‌گیرد.
"""

from graphmem import GraphMem, MemoryConfig, MemoryImportance
import os

class PersonalAssistant:
    def __init__(self, user_name: str):
        self.config = MemoryConfig(
            llm_provider="openai",
            llm_api_key=os.getenv("OPENAI_API_KEY"),
            llm_model="gpt-4o-mini",
            embedding_provider="openai",
            embedding_api_key=os.getenv("OPENAI_API_KEY"),
            embedding_model="text-embedding-3-small",
            
            # پایگاه داده شخصی ماندگار
            turso_db_path=f"~/.assistant/{user_name}.db",
            user_id=user_name,
            
            # تکامل حافظه
            evolution_enabled=True,
            decay_enabled=True,
            decay_half_life_days=90,  # نگهداری حافظه برای حدود ۳ ماه
        )
        self.memory = GraphMem(self.config, memory_id=f"personal_{user_name}", user_id=user_name)
        self.user_name = user_name
    
    def save_note(self, note: str, tags: list[str] = None):
        """ذخیره یادداشت شخصی."""
        self.memory.ingest(
            note,
            metadata={"type": "note", "tags": tags or []},
            importance=MemoryImportance.MEDIUM,
        )
    
    def save_article(self, title: str, content: str, url: str = None):
        """ذخیره مقاله با اهمیت بالا."""
        self.memory.ingest(
            f"مقاله: {title}\n\n{content}",
            metadata={"type": "article", "url": url},
            importance=MemoryImportance.HIGH,
        )
    
    def remember_fact(self, fact: str):
        """ذخیره یک واقعیت مهم."""
        self.memory.ingest(
            fact,
            importance=MemoryImportance.VERY_HIGH,
        )
    
    def ask(self, question: str) -> str:
        """پرسش هر سوالی از دستیار."""
        response = self.memory.query(question)
        return response.answer
    
    def daily_digest(self):
        """دریافت خلاصه‌ای از آنچه اخیراً آموخته‌اید."""
        response = self.memory.query(
            "مهم‌ترین چیزهایی که اخیراً یاد گرفتم چیست؟"
        )
        return response.answer
    
    def consolidate(self):
        """اجرای هفتگی برای تثبیت حافظه."""
        self.memory.evolve()

# نحوه استفاده
assistant = PersonalAssistant("alice")

# یادگیری
assistant.save_note("جلسه با باب: بررسی نقشه راه سه‌ماهه اول در سه‌شنبه آینده")
assistant.save_article(
    "آینده عامل‌های هوش مصنوعی",
    "عامل‌های هوش مصنوعی در حال توانمندتر شدن هستند...",
    url="https://example.com/ai-agents"
)
assistant.remember_fact("شناسه حساب AWS من 123456789 است")

# پرس‌وجو
print(assistant.ask("جلسه من با باب چه زمانی است؟"))
print(assistant.ask("شناسه حساب AWS من چیست؟"))
print(assistant.daily_digest())

# تثبیت هفتگی
assistant.consolidate()

```

### نمونه دوم: عامل پایگاه دانش سازمانی

```python
"""
پایگاه دانش سازمانی برای مدیریت:
- اسناد قوانین و سیاست‌ها
- اطلاعات کارمندان
- دستورالعمل‌های اجرایی
- مدیریت سوالات متداول (FAQ)
"""

from graphmem import GraphMem, MemoryConfig, MemoryImportance
from datetime import datetime
import os

class EnterpriseKB:
    def __init__(self, org_id: str):
        self.config = MemoryConfig(
            # استفاده از Azure OpenAI برای مصارف سازمانی
            llm_provider="azure_openai",
            llm_api_key=os.getenv("AZURE_OPENAI_KEY"),
            llm_api_base=os.getenv("AZURE_ENDPOINT"),
            azure_deployment="gpt-4",
            llm_model="gpt-4",
            azure_api_version="2024-02-15-preview",
            
            embedding_provider="azure_openai",
            embedding_api_key=os.getenv("AZURE_OPENAI_KEY"),
            embedding_api_base=os.getenv("AZURE_ENDPOINT"),
            azure_embedding_deployment="text-embedding-ada-002",
            embedding_model="text-embedding-ada-002",
            
            # Neo4j سازمانی
            neo4j_uri=os.getenv("NEO4J_URI"),
            neo4j_username="neo4j",
            neo4j_password=os.getenv("NEO4J_PASSWORD"),
            
            # Redis برای افزایش کارایی
            redis_url=os.getenv("REDIS_URL"),
            
            # ایزوله‌سازی سازمان
            user_id=org_id,
            
            # تکامل برای حل تناقضات
            evolution_enabled=True,
        )
        self.memory = GraphMem(self.config, memory_id=f"org_{org_id}", user_id=org_id)
        self.org_id = org_id
    
    def add_policy(self, policy_name: str, content: str, effective_date: str):
        """افزودن یا به‌روزرسانی یک سند قانون."""
        self.memory.ingest(
            f"قانون: {policy_name} (تاریخ اجرا: {effective_date})\n\n{content}",
            metadata={
                "type": "policy",
                "name": policy_name,
                "effective_date": effective_date,
            },
            importance=MemoryImportance.CRITICAL,
        )
        # اجرای تکامل برای جایگزینی نسخه‌های قدیمی قانون
        self.memory.evolve()
    
    def add_procedure(self, name: str, steps: list[str]):
        """افزودن دستورالعمل اجرایی."""
        content = f"دستورالعمل: {name}\n\n"
        content += "\n".join([f"{i+1}. {step}" for i, step in enumerate(steps)])
        
        self.memory.ingest(
            content,
            metadata={"type": "procedure", "name": name},
            importance=MemoryImportance.HIGH,
        )
    
    def add_employee_info(self, employee_data: dict):
        """افزودن اطلاعات کارمند."""
        content = f"""
        کارمند: {employee_data['name']}
        عنوان شغلی: {employee_data['title']}
        دپارتمان: {employee_data['department']}
        ایمیل: {employee_data['email']}
        مدیر مستقیم: {employee_data.get('manager', 'نامشخص')}
        تاریخ شروع: {employee_data.get('start_date', 'نامشخص')}
        """
        self.memory.ingest(
            content,
            metadata={"type": "employee", **employee_data},
            importance=MemoryImportance.MEDIUM,
        )
    
    def answer_question(self, question: str, department: str = None) -> dict:
        """پاسخ به سوال کارمند."""
        if department:
            question = f"[{department} department] {question}"
        
        response = self.memory.query(question)
        
        return {
            "answer": response.answer,
            "confidence": response.confidence,
            "sources": [n.name for n in response.nodes[:3]],
            "needs_escalation": response.confidence < 0.5,
        }
    
    def bulk_import(self, documents: list[dict]):
        """وارد کردن دسته‌ای اسناد از منابع مختلف."""
        self.memory.ingest_batch(
            documents,
            max_workers=20,
            aggressive=True,
            show_progress=True,
        )
        self.memory.evolve()

# نحوه استفاده
kb = EnterpriseKB("acme_corp")

# افزودن قوانین
kb.add_policy(
    "قانون دورکاری",
    "کارمندان می‌توانند تا ۳ روز در هفته دورکاری کنند...",
    "2024-01-01"
)

# به‌روزرسانی قانون (نسخه قدیمی به صورت خودکار جایگزین می‌شود)
kb.add_policy(
    "قانون دورکاری", 
    "کارمندان می‌توانند با تایید مدیر به صورت کاملاً دورکار فعالیت کنند...",
    "2024-06-01"
)

# افزودن دستورالعمل‌ها
kb.add_procedure("استرداد هزینه‌ها", [
    "ارسال گزارش هزینه ظرف ۳۰ روز",
    "ضمیمه کردن فاکتور برای موارد بالای ۲۵ دلار",
    "تایید مدیر برای موارد بالای ۵۰۰ دلار الزامی است",
    "پرداخت توسط بخش مالی ظرف ۵ روز کاری انجام می‌شود",
])

# پاسخ به سوالات
result = kb.answer_question("آیا می‌توانم از خانه کار کنم؟")
print(result["answer"])
# ← "بله، کارمندان می‌توانند با تایید مدیر به صورت کاملاً دورکار فعالیت کنند (اجرا از ژوئن ۲۰۲۴)"

```

---

## نتیجه‌گیری

پروژه GraphMem زیرساخت حافظه‌ای را فراهم می‌کند که پوشش‌های ساده روی LLM را به عامل‌های واقعی هوش مصنوعی تبدیل می‌کند. با مدیریت مواردی چون:

* ✅ **استخراج دانش** - استخراج خودکار موجودیت‌ها و روابط


* ✅ **ذخیره‌سازی معنایی** - بازنمایی دانش مبتنی بر گراف


* ✅ **بازیابی هوشمند** - استدلال چندمرحله‌ای و درک نام‌های مستعار


* ✅ **تکامل حافظه** - خودبهبوددهی از طریق تثبیت و زوال داده‌ها


* ✅ **اعتبار زمانی** - ردیابی زمان صحت واقعیت‌ها


* ✅ **رفع تناقضات** - ترجیح خودکار اطلاعات جدیدتر


* ✅ **مقیاس‌پذیری عملیاتی** - کش‌سازی با Redis و خوشه‌بندی با Neo4j



...شما می‌توانید تمرکز خود را بر توسعه منطق اصلی عامل قرار دهید و مدیریت حافظه را به GraphMem بسپارید.

---

## منابع

* **گیت‌هاب**: [https://github.com/Al-aminI/GraphMem](https://www.google.com/search?q=https://github.com/Al-aminI/GraphMem)

* **مخزن PyPI**: [https://pypi.org/project/agentic-graph-mem/](https://www.google.com/search?q=https://pypi.org/project/agentic-graph-mem/)

* **ثبت مسائل (Issues)**: [https://github.com/Al-aminI/GraphMem/issues](https://www.google.com/search?q=https://github.com/Al-aminI/GraphMem/issues)


---

*ساخته‌شده با ❤️ برای جامعه توسعه‌دهندگان عامل‌های هوش مصنوعی*


=================================


# 🧠 Building AI Agents with GraphMem

> **The definitive guide to building production AI agents with human-like memory**

GraphMem provides the memory layer that transforms stateless LLMs into intelligent agents capable of learning, remembering, and evolving over time.

---

## Table of Contents

1. [Why GraphMem for Agents?](#why-graphmem-for-agents)
2. [Quick Start](#quick-start)
3. [Core Concepts](#core-concepts)
4. [Agent Patterns](#agent-patterns)
5. [Production Architecture](#production-architecture)
6. [Advanced Features](#advanced-features)
7. [Best Practices](#best-practices)
8. [Complete Examples](#complete-examples)

---

## Why GraphMem for Agents?

### The Problem with Current Agent Memory

| Approach | Problem |
|----------|---------|
| **Context stuffing** | Token limits, expensive, no learning |
| **Simple vector DB** | No relationships, no temporal awareness |
| **Key-value stores** | No semantic understanding |
| **Traditional RAG** | Retrieves chunks, not knowledge |

### How GraphMem Solves This

```
┌─────────────────────────────────────────────────────────────────┐
│                    GRAPHMEM MEMORY SYSTEM                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  📥 INGEST           🧠 UNDERSTAND           📤 RETRIEVE        │
│  ─────────           ───────────             ──────────         │
│  • Text              • Entity extraction     • Semantic search  │
│  • Documents         • Relationship mapping  • Graph traversal  │
│  • Conversations     • Alias resolution      • Community query  │
│                      • Temporal tracking     • Multi-hop reason │
│                                                                  │
│  🔄 EVOLVE           ⚡ OPTIMIZE             🔒 ISOLATE         │
│  ────────            ──────────              ─────────          │
│  • Consolidation     • PageRank scoring      • Multi-tenant     │
│  • Decay old info    • Importance weighting  • User isolation   │
│  • Conflict resolve  • Caching (Redis)       • Session scoping  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Quick Start

### Installation

```bash
# Core (in-memory only - for testing)
pip install agentic-graph-mem

# With persistence (RECOMMENDED for production)
pip install "agentic-graph-mem[libsql]"
```

### Your First Memory-Enabled Agent

!!! warning "Important: Enable Persistence!"
    Without `turso_db_path`, your agent forgets everything on restart!

```python
from graphmem import GraphMem, MemoryConfig

# Initialize with your LLM provider
config = MemoryConfig(
    # LLM for extraction and querying
    llm_provider="openai",
    llm_api_key="sk-...",
    llm_model="gpt-4o-mini",
    
    # Embeddings for semantic search
    embedding_provider="openai",
    embedding_api_key="sk-...",
    embedding_model="text-embedding-3-small",
    
    # ✅ CRITICAL: Persistent storage!
    turso_db_path="agent_memory.db",
    
    # Enable self-evolution
    evolution_enabled=True,
    auto_evolve=True,  # Evolve after each ingestion
)

# Create memory instance with consistent memory_id AND user_id
memory = GraphMem(config, memory_id="my_agent", user_id="default")

# Your agent can now remember!
class SimpleAgent:
    def __init__(self, memory: GraphMem):
        self.memory = memory
    
    def learn(self, information: str):
        """Agent learns new information"""
        self.memory.ingest(information)
    
    def ask(self, question: str) -> str:
        """Agent answers from memory"""
        response = self.memory.query(question)
        return response.answer
    
    def reflect(self):
        """Agent consolidates and improves memory"""
        self.memory.evolve()

# Use the agent
agent = SimpleAgent(memory)
agent.learn("Tesla was founded in 2003. Elon Musk became CEO in 2008.")
agent.learn("SpaceX was founded by Elon Musk in 2002.")

print(agent.ask("What companies did Elon Musk found?"))
# → "Elon Musk founded SpaceX in 2002 and became CEO of Tesla in 2008."

agent.reflect()  # Consolidate knowledge about Elon Musk
```

---

## Core Concepts

### 1. The Three Pillars: Ingest → Query → Evolve

```python
# INGEST: Feed information to memory
memory.ingest("""
    Anthropic was founded in 2021 by Dario Amodei and Daniela Amodei.
    They previously worked at OpenAI. Anthropic created Claude.
""")

# QUERY: Ask questions
response = memory.query("Who founded Anthropic?")
print(response.answer)      # "Dario Amodei and Daniela Amodei founded Anthropic in 2021"
print(response.confidence)  # 0.95
print(response.context)     # Full context used for answering

# EVOLVE: Improve memory over time
events = memory.evolve()
for event in events:
    print(f"{event.evolution_type}: {event.description}")
    # CONSOLIDATION: Merged 3 mentions of "Anthropic" into 1 entity
    # DECAY: Archived 2 outdated relationships
```

### 2. Knowledge Graph Structure

GraphMem automatically builds a knowledge graph from your text:

```
                    ┌─────────────┐
                    │   Anthropic │
                    │ (Company)   │
                    └──────┬──────┘
                           │
           ┌───────────────┼───────────────┐
           │               │               │
           ▼               ▼               ▼
    ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
    │Dario Amodei │ │Daniela      │ │   Claude    │
    │  (Person)   │ │Amodei       │ │  (Product)  │
    │             │ │(Person)     │ │             │
    └─────────────┘ └─────────────┘ └─────────────┘
           │               │
           └───────┬───────┘
                   │
                   ▼
            ┌─────────────┐
            │   OpenAI    │
            │ (Company)   │
            └─────────────┘

Relationships:
• Dario Amodei --[founded]--> Anthropic (2021-present)
• Dario Amodei --[worked_at]--> OpenAI (past)
• Anthropic --[created]--> Claude
```

### 3. Exhaustive Extraction

GraphMem extracts **everything** from your text:

```python
memory.ingest("""
    In Q3 2024, Nvidia (NVDA) reported $35.1B revenue, up 94% YoY.
    CEO Jensen Huang announced Blackwell B200 shipping in early 2025.
    The company has 26,000 employees worldwide.
""")

# GraphMem extracts:
# ENTITIES:
# - Nvidia (Company) [aliases: NVDA, Nvidia Corporation]
# - Jensen Huang (Person) [aliases: J. Huang]
# - $35.1B (Amount)
# - 94% (Percentage)
# - Q3 2024 (Date)
# - Blackwell B200 (Product)
# - 26,000 (Number)
# - early 2025 (Date)

# RELATIONSHIPS:
# - Jensen Huang --[is CEO of]--> Nvidia
# - Nvidia --[reported revenue]--> $35.1B [valid: Q3 2024]
# - Nvidia --[achieved growth]--> 94%
# - Nvidia --[has employees]--> 26,000
# - Blackwell B200 --[ships in]--> early 2025
```

### 4. Source Chunk Preservation

Every entity preserves its **original source text** for accurate answers:

```python
# When you query, GraphMem provides:
response = memory.query("What is Nvidia's revenue?")

# The LLM sees:
# 1. ORIGINAL SOURCE TEXT (full context)
# 2. EXTRACTED ENTITIES (structured)
# 3. RELATIONSHIPS (connections)
# 4. COMMUNITY SUMMARIES (high-level understanding)
```

---

## Agent Patterns

### Pattern 1: Conversational Agent with Persistent Memory

```python
from graphmem import GraphMem, MemoryConfig, MemoryImportance

class ConversationalAgent:
    """Agent that remembers conversations across sessions."""
    
    def __init__(self, user_id: str):
        self.config = MemoryConfig(
            llm_provider="openai",
            llm_api_key="sk-...",
            llm_model="gpt-4o-mini",
            embedding_provider="openai",
            embedding_api_key="sk-...",
            embedding_model="text-embedding-3-small",
            
            # Persist to SQLite
            turso_db_path=f"memories/{user_id}.db",
            
            # User isolation
            user_id=user_id,
            
            # Evolution settings
            evolution_enabled=True,
            decay_enabled=True,
            decay_half_life_days=30,  # Forget unused info after ~30 days
        )
        self.memory = GraphMem(self.config, memory_id=f"agent_{user_id}", user_id=user_id)
        self.user_id = user_id
    
    def chat(self, user_message: str) -> str:
        """Process user message and generate response."""
        
        # 1. Store the user's message as a memory
        self.memory.ingest(
            f"User said: {user_message}",
            metadata={"type": "user_message", "timestamp": "now"},
            importance=MemoryImportance.MEDIUM,
        )
        
        # 2. Query memory for relevant context
        response = self.memory.query(user_message)
        
        # 3. Generate response (you'd use your own LLM here)
        agent_response = self._generate_response(user_message, response.context)
        
        # 4. Store the agent's response too
        self.memory.ingest(
            f"Agent responded: {agent_response}",
            metadata={"type": "agent_response"},
            importance=MemoryImportance.LOW,
        )
        
        return agent_response
    
    def _generate_response(self, query: str, context: str) -> str:
        """Generate response using LLM with memory context."""
        # Your LLM call here
        pass
    
    def end_session(self):
        """Consolidate memories at end of session."""
        self.memory.evolve()

# Usage
agent = ConversationalAgent(user_id="user_123")
agent.chat("My name is Alice and I work at Google.")
agent.chat("I'm interested in machine learning.")

# Later session...
agent.chat("What do you know about me?")
# → "You're Alice, you work at Google, and you're interested in machine learning."
```

### Pattern 2: Research Agent with Multi-Document Learning

```python
class ResearchAgent:
    """Agent that learns from multiple documents and answers complex questions."""
    
    def __init__(self):
        self.config = MemoryConfig(
            llm_provider="azure_openai",
            llm_api_key="...",
            llm_api_base="https://your-resource.openai.azure.com/",  # Azure endpoint
            azure_deployment="gpt-4",
            llm_model="gpt-4",
            azure_api_version="2024-02-15-preview",
            
            embedding_provider="azure_openai",
            embedding_api_key="...",
            embedding_api_base="https://your-resource.openai.azure.com/",  # Azure endpoint
            azure_embedding_deployment="text-embedding-ada-002",
            embedding_model="text-embedding-ada-002",
            
            # Use Neo4j for production graph storage
            neo4j_uri="neo4j+s://your-instance.databases.neo4j.io",
            neo4j_username="neo4j",
            neo4j_password="...",
            
            # Redis for caching
            redis_url="redis://...",
            
            # Aggressive evolution for research
            evolution_enabled=True,
            consolidation_threshold=0.75,  # More aggressive merging
        )
        self.memory = GraphMem(self.config, memory_id="research_agent", user_id="researcher")
    
    def ingest_documents(self, documents: list[dict]):
        """Ingest multiple documents efficiently."""
        result = self.memory.ingest_batch(
            documents,
            max_workers=20,  # Parallel processing
            aggressive=True,
            show_progress=True,
        )
        print(f"Ingested {result['documents_processed']} docs")
        print(f"Extracted {result['total_entities']} entities")
        print(f"Found {result['total_relationships']} relationships")
        
        # Evolve after batch ingestion
        self.memory.evolve()
    
    def research(self, question: str) -> dict:
        """Answer complex research questions."""
        response = self.memory.query(question)
        
        return {
            "answer": response.answer,
            "confidence": response.confidence,
            "sources": [n.name for n in response.nodes[:5]],
            "related_entities": [n.name for n in response.nodes],
            "context_tokens": len(response.context.split()),
        }
    
    def find_connections(self, entity_a: str, entity_b: str) -> str:
        """Find how two entities are connected."""
        query = f"How are {entity_a} and {entity_b} related or connected?"
        response = self.memory.query(query)
        return response.answer

# Usage
agent = ResearchAgent()

# Ingest research papers
papers = [
    {"id": "paper1", "content": "...paper about transformers..."},
    {"id": "paper2", "content": "...paper about attention mechanisms..."},
    {"id": "paper3", "content": "...paper about GPT architecture..."},
]
agent.ingest_documents(papers)

# Ask complex questions
result = agent.research("How did attention mechanisms evolve into modern LLMs?")
print(result["answer"])

# Find entity connections
print(agent.find_connections("Transformers", "GPT-4"))
```

### Pattern 3: Customer Support Agent with Conflict Resolution

```python
class SupportAgent:
    """Agent that handles customer support with up-to-date knowledge."""
    
    def __init__(self, company_id: str):
        self.config = MemoryConfig(
            llm_provider="openai",
            llm_api_key="sk-...",
            llm_model="gpt-4o",
            embedding_provider="openai",
            embedding_api_key="sk-...",
            embedding_model="text-embedding-3-small",
            
            turso_db_path=f"support/{company_id}.db",
            
            # Critical: Enable temporal validity for policy updates
            evolution_enabled=True,
            decay_enabled=True,
        )
        self.memory = GraphMem(self.config, memory_id="research_agent", user_id="researcher")
    
    def update_knowledge(self, content: str, importance: str = "HIGH"):
        """Update knowledge base with new information."""
        from graphmem import MemoryImportance
        
        importance_map = {
            "CRITICAL": MemoryImportance.CRITICAL,
            "HIGH": MemoryImportance.HIGH,
            "MEDIUM": MemoryImportance.MEDIUM,
            "LOW": MemoryImportance.LOW,
        }
        
        self.memory.ingest(
            content,
            importance=importance_map.get(importance, MemoryImportance.MEDIUM),
        )
        
        # CRITICAL: Evolve to resolve conflicts with old information
        # This uses priority-based decay to supersede outdated facts
        self.memory.evolve()
    
    def answer_ticket(self, ticket: str) -> dict:
        """Answer a support ticket using knowledge base."""
        response = self.memory.query(ticket)
        
        return {
            "answer": response.answer,
            "confidence": response.confidence,
            "needs_escalation": response.confidence < 0.6,
        }

# Usage
agent = SupportAgent("acme_corp")

# Initial knowledge
agent.update_knowledge("""
    Our return policy allows returns within 30 days of purchase.
    Refunds are processed within 5-7 business days.
""")

# Policy update - GraphMem will resolve the conflict!
agent.update_knowledge("""
    POLICY UPDATE (2024): Our new return policy allows returns within 60 days.
    Refunds are now processed within 2-3 business days.
""", importance="CRITICAL")

# Agent now answers with updated info
result = agent.answer_ticket("What's your return policy?")
print(result["answer"])
# → "Our return policy allows returns within 60 days. Refunds are processed within 2-3 business days."
# (Old 30-day policy was automatically superseded)
```

### Pattern 4: Multi-Tenant SaaS Agent

```python
class MultiTenantAgent:
    """Agent that serves multiple customers with isolated memories."""
    
    def __init__(self, base_config: dict):
        self.base_config = base_config
        self.memories = {}  # tenant_id -> GraphMem
    
    def get_memory(self, tenant_id: str) -> GraphMem:
        """Get or create memory for a tenant."""
        if tenant_id not in self.memories:
            config = MemoryConfig(
                **self.base_config,
                
                # CRITICAL: Tenant isolation
                user_id=tenant_id,
                memory_id=f"tenant_{tenant_id}",
                
                # Each tenant gets own database
                turso_db_path=f"tenants/{tenant_id}/memory.db",
            )
            self.memories[tenant_id] = GraphMem(config, memory_id=f"tenant_{tenant_id}", user_id=tenant_id)
        
        return self.memories[tenant_id]
    
    def ingest(self, tenant_id: str, content: str):
        """Ingest content for a specific tenant."""
        memory = self.get_memory(tenant_id)
        memory.ingest(content)
    
    def query(self, tenant_id: str, question: str) -> str:
        """Query a specific tenant's memory."""
        memory = self.get_memory(tenant_id)
        response = memory.query(question)
        return response.answer

# Usage
base_config = {
    "llm_provider": "openai",
    "llm_api_key": "sk-...",
    "llm_model": "gpt-4o-mini",
    "embedding_provider": "openai",
    "embedding_api_key": "sk-...",
    "embedding_model": "text-embedding-3-small",
}

agent = MultiTenantAgent(base_config)

# Each tenant's data is completely isolated
agent.ingest("acme", "Acme's product costs $99.")
agent.ingest("globex", "Globex's product costs $149.")

print(agent.query("acme", "What's the product price?"))   # → "$99"
print(agent.query("globex", "What's the product price?")) # → "$149"
```

---

## Production Architecture

### Recommended Stack

```
┌─────────────────────────────────────────────────────────────────┐
│                    PRODUCTION DEPLOYMENT                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    APPLICATION LAYER                      │    │
│  │                                                          │    │
│  │   FastAPI / Flask / Django                               │    │
│  │       ↓                                                  │    │
│  │   GraphMem Instance (per request or singleton)           │    │
│  └─────────────────────────────────────────────────────────┘    │
│                            ↓                                     │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                     STORAGE LAYER                         │    │
│  │                                                          │    │
│  │  Neo4j Aura (Graph)  ←→  Redis Cloud (Cache)            │    │
│  │         ↑                        ↑                       │    │
│  │         └────────────────────────┘                       │    │
│  │                    ↓                                     │    │
│  │            Turso (Backup/Vectors)                        │    │
│  └─────────────────────────────────────────────────────────┘    │
│                            ↓                                     │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    LLM PROVIDERS                          │    │
│  │                                                          │    │
│  │  OpenAI  |  Azure OpenAI  |  Anthropic  |  Local LLMs   │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### FastAPI Integration Example

```python
from fastapi import FastAPI, Depends, HTTPException
from pydantic import BaseModel
from graphmem import GraphMem, MemoryConfig
from functools import lru_cache

app = FastAPI()

# Singleton memory instance
@lru_cache()
def get_memory() -> GraphMem:
    config = MemoryConfig(
        llm_provider="openai",
        llm_api_key=os.getenv("OPENAI_API_KEY"),
        llm_model="gpt-4o-mini",
        embedding_provider="openai",
        embedding_api_key=os.getenv("OPENAI_API_KEY"),
        embedding_model="text-embedding-3-small",
        
        # Production storage
        neo4j_uri=os.getenv("NEO4J_URI"),
        neo4j_username=os.getenv("NEO4J_USER"),
        neo4j_password=os.getenv("NEO4J_PASSWORD"),
        
        # Redis caching
        redis_url=os.getenv("REDIS_URL"),
        
        evolution_enabled=True,
    )
    return GraphMem(config, memory_id="api_agent", user_id="api_user")

class IngestRequest(BaseModel):
    content: str
    metadata: dict = {}

class QueryRequest(BaseModel):
    question: str

class QueryResponse(BaseModel):
    answer: str
    confidence: float
    sources: list[str]

@app.post("/ingest")
async def ingest(request: IngestRequest, memory: GraphMem = Depends(get_memory)):
    result = memory.ingest(request.content, metadata=request.metadata)
    return {"status": "success", "entities": result["entities"]}

@app.post("/query", response_model=QueryResponse)
async def query(request: QueryRequest, memory: GraphMem = Depends(get_memory)):
    response = memory.query(request.question)
    return QueryResponse(
        answer=response.answer,
        confidence=response.confidence,
        sources=[n.name for n in response.nodes[:5]],
    )

@app.post("/evolve")
async def evolve(memory: GraphMem = Depends(get_memory)):
    events = memory.evolve()
    return {"events": len(events)}

@app.get("/stats")
async def stats(memory: GraphMem = Depends(get_memory)):
    return memory.get_stats()
```

### High-Throughput Batch Ingestion

```python
import asyncio
from concurrent.futures import ThreadPoolExecutor

class HighThroughputIngestion:
    """Ingest millions of documents efficiently."""
    
    def __init__(self, memory: GraphMem):
        self.memory = memory
    
    def ingest_large_dataset(
        self,
        documents: list[dict],
        batch_size: int = 1000,
        max_workers: int = 20,
    ):
        """Ingest documents in batches with progress tracking."""
        total = len(documents)
        processed = 0
        
        for i in range(0, total, batch_size):
            batch = documents[i:i+batch_size]
            
            result = self.memory.ingest_batch(
                batch,
                max_workers=max_workers,
                aggressive=True,
                show_progress=True,
                rebuild_communities=False,  # Defer until end
            )
            
            processed += result["documents_processed"]
            print(f"Progress: {processed}/{total} ({100*processed/total:.1f}%)")
        
        # Rebuild communities once at the end
        print("Building communities...")
        self.memory.evolve()
        
        return {"total_processed": processed}

# Usage
ingestion = HighThroughputIngestion(memory)
ingestion.ingest_large_dataset(
    documents=large_dataset,  # Millions of docs
    batch_size=1000,
    max_workers=20,
)
```

---

## Advanced Features

### 1. Temporal Queries (Point-in-Time)

```python
from datetime import datetime

# Ingest historical data
memory.ingest("""
    Steve Jobs was CEO of Apple from 1997 to 2011.
    Tim Cook became CEO in August 2011 and is still CEO today.
""")

# Query about the past
response = memory.query("Who was CEO of Apple in 2005?")
print(response.answer)  # → "Steve Jobs was CEO of Apple from 1997 to 2011"

# Query about the present
response = memory.query("Who is the current CEO of Apple?")
print(response.answer)  # → "Tim Cook has been CEO since August 2011"
```

### 2. Alias-Aware Retrieval

```python
# GraphMem automatically extracts and uses aliases
memory.ingest("""
    Dr. Alexander Chen, also known as "The Quantum Pioneer", 
    founded Quantum AI Labs. Alex Chen received his PhD from MIT.
""")

# All of these work:
memory.query("What did Dr. Chen do?")
memory.query("Who is Alexander Chen?")
memory.query("Tell me about The Quantum Pioneer")
memory.query("What did Alex Chen study?")
# All return information about the same person!
```

### 3. Multi-Hop Reasoning

```python
# GraphMem can traverse relationships to answer complex questions
memory.ingest("Apple was founded by Steve Jobs.")
memory.ingest("Steve Jobs also founded NeXT.")
memory.ingest("NeXT was acquired by Apple in 1997.")
memory.ingest("Tim Cook worked at Compaq before Apple.")

# Multi-hop question
response = memory.query("What's the connection between NeXT and Tim Cook?")
# GraphMem traverses: NeXT → Steve Jobs → Apple → Tim Cook
print(response.answer)
# → "NeXT was founded by Steve Jobs and acquired by Apple in 1997. 
#    Tim Cook currently works at Apple as CEO."
```

### 4. Conflict Resolution via Evolution

```python
# Initial fact
memory.ingest("The company has 1,000 employees.")

# Updated fact
memory.ingest("BREAKING: The company now has 2,500 employees after expansion.")

# Evolution automatically resolves conflicts
memory.evolve()

# Queries return the newer information
response = memory.query("How many employees does the company have?")
print(response.answer)  # → "2,500 employees (after expansion)"
```

### 5. Importance-Based Memory

```python
from graphmem import MemoryImportance

# Critical information (never decays)
memory.ingest(
    "Customer's allergies: peanuts, shellfish",
    importance=MemoryImportance.CRITICAL,
)

# Normal information (may decay over time)
memory.ingest(
    "Customer mentioned they like coffee",
    importance=MemoryImportance.LOW,
)

# After evolution, low-importance unused memories decay
# Critical memories are always preserved
```

---

## Best Practices

### 1. Structure Your Ingestion

```python
# ❌ BAD: Unstructured dumps
memory.ingest("stuff happened today blah blah...")

# ✅ GOOD: Clear, factual statements
memory.ingest("""
    On January 15, 2024, Acme Corp announced Q4 earnings:
    - Revenue: $5.2 billion (up 23% YoY)
    - Net income: $890 million
    - CEO Jane Smith attributed growth to AI products
""")
```

### 2. Use Batch Ingestion for Large Datasets

```python
# ❌ BAD: Sequential ingestion
for doc in documents:
    memory.ingest(doc["content"])  # Slow!

# ✅ GOOD: Batch ingestion
memory.ingest_batch(
    documents,
    max_workers=20,
    aggressive=True,
)
```

### 3. Evolve Regularly

```python
# Option 1: Auto-evolve (simple)
config = MemoryConfig(..., auto_evolve=True)

# Option 2: Explicit evolution (more control)
# After ingestion sessions
memory.ingest_batch(docs)
memory.evolve()

# Or on a schedule (e.g., daily)
import schedule
schedule.every().day.at("02:00").do(memory.evolve)
```

### 4. Use Appropriate Storage

| Use Case | Recommended Storage |
|----------|-------------------|
| **Development/Testing** | Turso (SQLite) - zero setup |
| **Single-server production** | Turso + Redis |
| **Multi-server production** | Neo4j + Redis |
| **Enterprise/High-scale** | Neo4j Aura + Redis Cloud |

### 5. Handle Errors Gracefully

```python
from graphmem.core.exceptions import IngestionError, QueryError

try:
    memory.ingest(content)
except IngestionError as e:
    logger.error(f"Ingestion failed: {e}")
    # Retry or queue for later

try:
    response = memory.query(question)
except QueryError as e:
    logger.error(f"Query failed: {e}")
    # Fallback to default response
```

---

## Complete Examples

### Example 1: Personal Knowledge Assistant

```python
"""
A personal knowledge assistant that learns from your notes,
articles, and conversations.
"""

from graphmem import GraphMem, MemoryConfig, MemoryImportance
import os

class PersonalAssistant:
    def __init__(self, user_name: str):
        self.config = MemoryConfig(
            llm_provider="openai",
            llm_api_key=os.getenv("OPENAI_API_KEY"),
            llm_model="gpt-4o-mini",
            embedding_provider="openai",
            embedding_api_key=os.getenv("OPENAI_API_KEY"),
            embedding_model="text-embedding-3-small",
            
            # Persistent personal database
            turso_db_path=f"~/.assistant/{user_name}.db",
            user_id=user_name,
            
            # Memory evolution
            evolution_enabled=True,
            decay_enabled=True,
            decay_half_life_days=90,  # Keep memories for ~3 months
        )
        self.memory = GraphMem(self.config, memory_id=f"personal_{user_name}", user_id=user_name)
        self.user_name = user_name
    
    def save_note(self, note: str, tags: list[str] = None):
        """Save a personal note."""
        self.memory.ingest(
            note,
            metadata={"type": "note", "tags": tags or []},
            importance=MemoryImportance.MEDIUM,
        )
    
    def save_article(self, title: str, content: str, url: str = None):
        """Save an article with high importance."""
        self.memory.ingest(
            f"Article: {title}\n\n{content}",
            metadata={"type": "article", "url": url},
            importance=MemoryImportance.HIGH,
        )
    
    def remember_fact(self, fact: str):
        """Save an important fact."""
        self.memory.ingest(
            fact,
            importance=MemoryImportance.VERY_HIGH,
        )
    
    def ask(self, question: str) -> str:
        """Ask your assistant anything."""
        response = self.memory.query(question)
        return response.answer
    
    def daily_digest(self):
        """Get a summary of what you learned recently."""
        response = self.memory.query(
            "What are the most important things I learned recently?"
        )
        return response.answer
    
    def consolidate(self):
        """Run weekly to consolidate memories."""
        self.memory.evolve()

# Usage
assistant = PersonalAssistant("alice")

# Learning
assistant.save_note("Meeting with Bob: Discuss Q1 roadmap next Tuesday")
assistant.save_article(
    "The Future of AI Agents",
    "AI agents are becoming more capable...",
    url="https://example.com/ai-agents"
)
assistant.remember_fact("My AWS account ID is 123456789")

# Querying
print(assistant.ask("When is my meeting with Bob?"))
print(assistant.ask("What's my AWS account ID?"))
print(assistant.daily_digest())

# Weekly consolidation
assistant.consolidate()
```

### Example 2: Enterprise Knowledge Base Agent

```python
"""
Enterprise knowledge base that handles:
- Policy documents
- Employee information
- Procedure manuals
- FAQ management
"""

from graphmem import GraphMem, MemoryConfig, MemoryImportance
from datetime import datetime
import os

class EnterpriseKB:
    def __init__(self, org_id: str):
        self.config = MemoryConfig(
            # Use Azure OpenAI for enterprise
            llm_provider="azure_openai",
            llm_api_key=os.getenv("AZURE_OPENAI_KEY"),
            llm_api_base=os.getenv("AZURE_ENDPOINT"),  # e.g., "https://your-resource.openai.azure.com/"
            azure_deployment="gpt-4",
            llm_model="gpt-4",
            azure_api_version="2024-02-15-preview",
            
            embedding_provider="azure_openai",
            embedding_api_key=os.getenv("AZURE_OPENAI_KEY"),
            embedding_api_base=os.getenv("AZURE_ENDPOINT"),  # Same endpoint for embeddings
            azure_embedding_deployment="text-embedding-ada-002",
            embedding_model="text-embedding-ada-002",
            
            # Enterprise Neo4j
            neo4j_uri=os.getenv("NEO4J_URI"),
            neo4j_username="neo4j",
            neo4j_password=os.getenv("NEO4J_PASSWORD"),
            
            # Redis for performance
            redis_url=os.getenv("REDIS_URL"),
            
            # Org isolation
            user_id=org_id,
            
            # Evolution for conflict resolution
            evolution_enabled=True,
        )
        self.memory = GraphMem(self.config, memory_id=f"org_{org_id}", user_id=org_id)
        self.org_id = org_id
    
    def add_policy(self, policy_name: str, content: str, effective_date: str):
        """Add or update a policy document."""
        self.memory.ingest(
            f"POLICY: {policy_name} (Effective: {effective_date})\n\n{content}",
            metadata={
                "type": "policy",
                "name": policy_name,
                "effective_date": effective_date,
            },
            importance=MemoryImportance.CRITICAL,
        )
        # Evolve to supersede old policy versions
        self.memory.evolve()
    
    def add_procedure(self, name: str, steps: list[str]):
        """Add a procedure manual."""
        content = f"PROCEDURE: {name}\n\n"
        content += "\n".join([f"{i+1}. {step}" for i, step in enumerate(steps)])
        
        self.memory.ingest(
            content,
            metadata={"type": "procedure", "name": name},
            importance=MemoryImportance.HIGH,
        )
    
    def add_employee_info(self, employee_data: dict):
        """Add employee information."""
        content = f"""
        EMPLOYEE: {employee_data['name']}
        Title: {employee_data['title']}
        Department: {employee_data['department']}
        Email: {employee_data['email']}
        Reports to: {employee_data.get('manager', 'N/A')}
        Start date: {employee_data.get('start_date', 'N/A')}
        """
        self.memory.ingest(
            content,
            metadata={"type": "employee", **employee_data},
            importance=MemoryImportance.MEDIUM,
        )
    
    def answer_question(self, question: str, department: str = None) -> dict:
        """Answer an employee's question."""
        # Add department context if provided
        if department:
            question = f"[{department} department] {question}"
        
        response = self.memory.query(question)
        
        return {
            "answer": response.answer,
            "confidence": response.confidence,
            "sources": [n.name for n in response.nodes[:3]],
            "needs_escalation": response.confidence < 0.5,
        }
    
    def bulk_import(self, documents: list[dict]):
        """Bulk import documents from various sources."""
        self.memory.ingest_batch(
            documents,
            max_workers=20,
            aggressive=True,
            show_progress=True,
        )
        self.memory.evolve()

# Usage
kb = EnterpriseKB("acme_corp")

# Add policies
kb.add_policy(
    "Remote Work Policy",
    "Employees may work remotely up to 3 days per week...",
    "2024-01-01"
)

# Update policy (old version auto-superseded)
kb.add_policy(
    "Remote Work Policy", 
    "Employees may work fully remote with manager approval...",
    "2024-06-01"
)

# Add procedures
kb.add_procedure("Expense Reimbursement", [
    "Submit expense report within 30 days",
    "Include receipts for items over $25",
    "Manager approval required for items over $500",
    "Finance processes within 5 business days",
])

# Answer questions
result = kb.answer_question("Can I work from home?")
print(result["answer"])
# → "Yes, employees may work fully remote with manager approval (policy effective June 2024)"
```

---

## Conclusion

GraphMem provides the memory infrastructure that transforms simple LLM wrappers into true AI agents. By handling:

- ✅ **Knowledge extraction** - Automatic entity and relationship extraction
- ✅ **Semantic storage** - Graph-based knowledge representation
- ✅ **Intelligent retrieval** - Multi-hop reasoning and alias awareness
- ✅ **Memory evolution** - Self-improving through consolidation and decay
- ✅ **Temporal validity** - Track when facts were true
- ✅ **Conflict resolution** - Automatically prefer newer information
- ✅ **Production scaling** - Redis caching, Neo4j clustering

...you can focus on building the agent logic while GraphMem handles the memory.

---

## Resources

- **GitHub**: https://github.com/Al-aminI/GraphMem
- **PyPI**: https://pypi.org/project/agentic-graph-mem/
- **Issues**: https://github.com/Al-aminI/GraphMem/issues

---

*Built with ❤️ for the AI Agent community*

