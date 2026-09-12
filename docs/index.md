ترجمه فارسی و دقیق فایل `index.md` (نخستین فایل از پوشه مستندات) آماده شد. این متن را می‌توانید مستقیماً در فایل `index.md` پروژه قرار دهید:

---

# 🧠 GraphMem-Go

## **مغز انسان برای عامل‌های هوش مصنوعی شما**

> **"حافظه، گنجینه و نگهبان همه چیز است."** — سیسرون

پروژه GraphMem **نخستین سیستم حافظه‌ای است که همانند مغز انسان فکر می‌کند**. این سیستم صرفاً داده‌ها را ذخیره نمی‌کند؛ بلکه دقیقاً مانند حافظه زیستی **فراموش می‌کند**، **تثبیت و یکپارچه می‌سازد**، **اولویه‌بندی می‌کند** و **تکامل می‌یابد**.

**این آینده عامل‌های هوش مصنوعی سازمانی است.**

---

## 🧬 چرا GraphMem همه چیز را تغییر می‌دهد؟

### مشکل حافظه‌های فعلی هوش مصنوعی

همه عامل‌های هوش مصنوعی عملیاتی با بحران یکسانی مواجه هستند:

```text
روز ۱:    "مدیرعامل کیست؟" ← "ایلان ماسک" ✅
روز ۱۰۰:  پنجره بافت (Context Window): سرریز و انفجار 💥
روز ۳۶۵:  "مدیرعامل کیست؟" ← "جان... یا شاید جین... شایدم ایلان؟" 🤯

```

**پایگاه‌های داده برداری فراموش نمی‌کنند.** آن‌ها اطلاعات زائد را تجمع می‌دهند تا زمانی که عامل شما در داده‌های غیرمرتبط، متناقض و قدیمی غرق شود.

### راهکار GraphMem: حافظه‌ای که فکر می‌کند

پروژه GraphMem **چهار رکن حافظه انسانی** را پیاده‌سازی می‌کند:

| مغز انسان | GraphMem | چرا اهمیت دارد؟ |
| --- | --- | --- |
| 🧠 **منحنی فراموشی** | زوال حافظه (Memory Decay) | حافظه‌های غیرمرتبط به صورت طبیعی محو می‌شوند

 |
| 🔗 **شبکه‌های عصبی** | گراف دانش (Knowledge Graph) | روابط میان مفاهیم درک می‌شود

 |
| ⭐ **وزن‌دهی اهمیت** | مرکزیت PageRank | مفاهیم کلیدی (مانند ایلان ماسک) > مفاهیم حاشیه‌ای

 |
| ⏰ **حافظه رویدادی** | اعتبار زمانی (Temporal Validity) | "مدیرعامل در سال ۲۰۱۵" در برابر "مدیرعامل فعلی"

 |

---

## 🚀 شروع سریع

### نصب

```bash
go get github.com/flancast90/GraphMem-go

```

### استفاده پایه

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/flancast90/GraphMem-go/pkg/graphmem"
)

func main() {
    // ساخت پیکربندی (خواندن از متغیرهای محیطی)
    config := graphmem.NewConfig()
    config.LLMAPIKey = os.Getenv("OPENAI_API_KEY")

    // ساخت نمونه GraphMem
    gm, err := graphmem.New(config,
        graphmem.WithUserID("my_agent"),
        graphmem.WithAutoEvolve(true),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer gm.Close()

    // تمام شد؛ فقط ۳ متد اصلی:
    result, _ := gm.Ingest("تسلا توسط مدیرعامل ایلان ماسک رهبری می‌شود...")  // ← استخراج دانش
    response, _ := gm.Query("مدیرعامل کیست؟")                              // ← پرسش و پاسخ
    gm.Evolve()                                                          // ← پختگی و تکامل حافظه

    fmt.Printf("%d موجودیت استخراج شد\n", result.Entities)
    fmt.Println("پاسخ:", response.Answer)
}

```

### همراه با ماندگاری داده‌ها (Neo4j)

```go
config := graphmem.NewConfig()
config.LLMAPIKey = os.Getenv("OPENAI_API_KEY")
config.Neo4jURI = "bolt://localhost:7687"
config.Neo4jUser = "neo4j"
config.Neo4jPassword = "password"

gm, err := graphmem.New(config,
    graphmem.WithUserID("my_agent"),
)
// داده‌ها بین اجرای مجدد برنامه باقی می‌مانند!

```

### همراه با بافر کش (Redis)

```go
config := graphmem.NewConfig()
config.RedisURL = "redis://localhost:6379"

gm, err := graphmem.New(config,
    graphmem.WithUserID("my_agent"),
)
// پرس‌وجوها برای افزایش کارایی کش می‌شوند!

```

### استفاده از سرویس‌دهندگان مختلف LLM

```go
// OpenAI (پیش‌فرض)
config := graphmem.NewConfig()
config.LLMProvider = "openai"
config.LLMAPIKey = os.Getenv("OPENAI_API_KEY")
config.LLMModel = "gpt-4o-mini"

// Anthropic Claude
config.LLMProvider = "anthropic"
config.LLMAPIKey = os.Getenv("ANTHROPIC_API_KEY")
config.LLMModel = "claude-3-haiku-20240307"

// Azure OpenAI
config.LLMProvider = "azure_openai"
config.AzureOpenAIEndpoint = "https://your-resource.openai.azure.com"
config.AzureOpenAIDeployment = "gpt-4"
config.LLMAPIKey = os.Getenv("AZURE_OPENAI_API_KEY")

// Ollama محلی
config.LLMProvider = "ollama"
config.OllamaBaseURL = "http://localhost:11434"
config.LLMModel = "llama3.2"

```

---

## 🎯 ویژگی‌های تحول‌آفرین

### حافظه مبتنی بر نقطه زمانی

پرس‌وجو در گذشته: *"در سال ۲۰۱۵ مدیرعامل چه کسی بود؟"*

[بیشتر بدانید ←](https://www.google.com/search?q=concepts/temporal.md)

### گراف دانش

استخراج خودکار موجودیت‌ها و نگاشت روابط

[بیشتر بدانید ←](https://www.google.com/search?q=concepts/knowledge-graph.md)

### خودتکاملی

حافظه‌ای که یکپارچه می‌شود، زوال می‌یابد و بهبود می‌یابد

[بیشتر بدانید ←](https://www.google.com/search?q=concepts/evolution.md)

### جداسازی چندمستأجره (Multi-Tenant Isolation)

جداسازی کامل داده‌ها برای مصارف سازمانی

[بیشتر بدانید ←](https://www.google.com/search?q=concepts/multi-tenancy.md)

---

## 📊 عملکرد

| معیار | RAG معمولی | GraphMem | مزیت |
| --- | --- | --- | --- |
| **۱,۰۰۰ گفتگو** | 💥 سرریز بافت | ✅ محدود و مدیریت‌شده | پشتیبانی از رشد داده

 |
| **۱۰,۰۰۰ موجودیت** | O(n) = ۲.۳ ثانیه | O(1) = ۵۰ میلی‌ثانیه | **۴۶ برابر سریع‌تر**<br> |
| **تاریخچه ۱ ساله** | ۳,۶۵۰ ورودی | حدود ۱۰ ورودی تثبیت‌شده | **۹۷٪ کاهش حجم**<br> |
| **تناقض موجودیت‌ها** | تکراری‌ها باقی می‌مانند | حل خودکار تناقضات | داده‌های تمیز

 |
| **پرس‌وجوی زمانی** | ❌ غیرممکن | ✅ نیتیو و داخلی | قابلیت منحصر‌به‌فرد

 |

---

## 🐳 راه اندازی با داکر

```bash
# اجرای تمام سرویس‌ها (Neo4j, Redis, LibSQL)
make services-up

# اجرای تست‌های یکپارچه‌سازی
make test-integration

# متوقف کردن سرویس‌ها
make services-down

```

برای مشاهده تنظیمات کامل به فایل [docker-compose.yml](https://www.google.com/search?q=../docker-compose.yml) مراجعه کنید.

---

## 📚 مستندات

* **[شروع کار](https://www.google.com/search?q=getting-started/installation.md)** - راهنمای نصب، شروع سریع و پیکربندی


* **[مفاهیم پایه](https://www.google.com/search?q=concepts/overview.md)** - درک نحوه عملکرد GraphMem


* **[ساخت عامل‌ها](https://www.google.com/search?q=agents/guide.md)** - راهنمای جامع ساخت عامل‌های هوش مصنوعی


* **[محیط عملیاتی](https://www.google.com/search?q=production/architecture.md)** - استقرار در مقیاس وسیع با اطمینان خاطر



---

## 🤝 مشارکت

ما در حال ساخت آینده حافظه هوش مصنوعی هستیم. به ما بپیوندید!

* 🐛 [گزارش باگ‌ها](https://www.google.com/search?q=https://github.com/flancast90/GraphMem-go/issues)

* 💡 [درخواست ویژگی‌های جدید](https://www.google.com/search?q=https://github.com/flancast90/GraphMem-go/issues)

* 🔀 [ارسال PR ها](https://www.google.com/search?q=https://github.com/flancast90/GraphMem-go/pulls)


---

**GraphMem-Go** - پیاده‌سازی GraphMem به زبان Go

*"به عامل‌های هوش مصنوعی خود حافظه‌ای را بدهید که شایسته آن هستند."*


================================

# 🧠 GraphMem-Go

## **The Human Brain for Your AI Agents**

<p align="center">
  <a href="https://goreportcard.com/report/github.com/flancast90/GraphMem-go"><img src="https://goreportcard.com/badge/github.com/flancast90/GraphMem-go" alt="Go Report Card"></a>
  <a href="https://pkg.go.dev/github.com/flancast90/GraphMem-go"><img src="https://pkg.go.dev/badge/github.com/flancast90/GraphMem-go.svg" alt="Go Reference"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT"></a>
  <a href="https://github.com/flancast90/GraphMem-go"><img src="https://img.shields.io/badge/github-flancast90/GraphMem--go-blue.svg" alt="GitHub"></a>
</p>

> **"Memory is the treasury and guardian of all things."** — Cicero

GraphMem is the **first memory system that thinks like a human brain**. It doesn't just store data—it **forgets**, **consolidates**, **prioritizes**, and **evolves** exactly like biological memory does.

**This is the future of enterprise AI agents.**

---

## 🧬 Why GraphMem Changes Everything

### The Problem with Current AI Memory

Every production AI agent faces the same crisis:

```
Day 1:     "Who is the CEO?" → "Elon Musk" ✅
Day 100:   Context window: OVERFLOW 💥
Day 365:   "Who is the CEO?" → "John... or was it Jane... maybe Elon?" 🤯
```

**Vector databases don't forget.** They accumulate garbage until your agent drowns in irrelevant, conflicting, outdated information.

### The GraphMem Solution: Memory That Thinks

GraphMem implements the **four pillars of human memory**:

| Human Brain | GraphMem | Why It Matters |
|-------------|----------|----------------|
| 🧠 **Forgetting Curve** | Memory Decay | Irrelevant memories fade naturally |
| 🔗 **Neural Networks** | Knowledge Graph | Relationships between concepts |
| ⭐ **Importance Weighting** | PageRank Centrality | Hub concepts (Elon Musk) > peripheral ones |
| ⏰ **Episodic Memory** | Temporal Validity | "CEO in 2015" vs "CEO now" |

---

## 🚀 Quick Start

### Installation

```bash
go get github.com/flancast90/GraphMem-go
```

### Basic Usage

```go
package main

import (
    "fmt"
    "log"
    "os"

    "github.com/flancast90/GraphMem-go/pkg/graphmem"
)

func main() {
    // Create configuration (reads from environment)
    config := graphmem.NewConfig()
    config.LLMAPIKey = os.Getenv("OPENAI_API_KEY")

    // Create GraphMem instance
    gm, err := graphmem.New(config,
        graphmem.WithUserID("my_agent"),
        graphmem.WithAutoEvolve(true),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer gm.Close()

    // That's it. 3 methods:
    result, _ := gm.Ingest("Tesla is led by CEO Elon Musk...")  // ← Extract knowledge
    response, _ := gm.Query("Who is the CEO?")                   // ← Ask questions
    gm.Evolve()                                                  // ← Let memory mature

    fmt.Printf("Extracted %d entities\n", result.Entities)
    fmt.Println("Answer:", response.Answer)
}
```

### With Persistence (Neo4j)

```go
config := graphmem.NewConfig()
config.LLMAPIKey = os.Getenv("OPENAI_API_KEY")
config.Neo4jURI = "bolt://localhost:7687"
config.Neo4jUser = "neo4j"
config.Neo4jPassword = "password"

gm, err := graphmem.New(config,
    graphmem.WithUserID("my_agent"),
)
// Data persists between restarts!
```

### With Caching (Redis)

```go
config := graphmem.NewConfig()
config.RedisURL = "redis://localhost:6379"

gm, err := graphmem.New(config,
    graphmem.WithUserID("my_agent"),
)
// Queries are cached for performance!
```

### Using Different LLM Providers

```go
// OpenAI (default)
config := graphmem.NewConfig()
config.LLMProvider = "openai"
config.LLMAPIKey = os.Getenv("OPENAI_API_KEY")
config.LLMModel = "gpt-4o-mini"

// Anthropic Claude
config.LLMProvider = "anthropic"
config.LLMAPIKey = os.Getenv("ANTHROPIC_API_KEY")
config.LLMModel = "claude-3-haiku-20240307"

// Azure OpenAI
config.LLMProvider = "azure_openai"
config.AzureOpenAIEndpoint = "https://your-resource.openai.azure.com"
config.AzureOpenAIDeployment = "gpt-4"
config.LLMAPIKey = os.Getenv("AZURE_OPENAI_API_KEY")

// Local Ollama
config.LLMProvider = "ollama"
config.OllamaBaseURL = "http://localhost:11434"
config.LLMModel = "llama3.2"
```

---

## 🎯 Revolutionary Features

### Point-in-Time Memory

Query the past: *"Who was CEO in 2015?"*

[Learn more →](concepts/temporal.md)

### Knowledge Graph

Automatic entity extraction and relationship mapping

[Learn more →](concepts/knowledge-graph.md)

### Self-Evolution

Memory that consolidates, decays, and improves

[Learn more →](concepts/evolution.md)

### Multi-Tenant Isolation

Complete data separation for enterprise

[Learn more →](concepts/multi-tenancy.md)

---

## 📊 Performance

| Metric | Naive RAG | GraphMem | Advantage |
|--------|-----------|----------|-----------|
| **1K conversations** | 💥 Context overflow | ✅ Bounded | Handles growth |
| **10K entities** | O(n) = 2.3s | O(1) = 50ms | **46x faster** |
| **1 year history** | 3,650 entries | ~100 consolidated | **97% reduction** |
| **Entity conflicts** | Duplicates | Auto-resolved | Clean data |
| **Temporal queries** | ❌ Impossible | ✅ Native | Unique capability |

---

## 🐳 Docker Setup

```bash
# Start all services (Neo4j, Redis, LibSQL)
make services-up

# Run integration tests
make test-integration

# Stop services
make services-down
```

See [docker-compose.yml](../docker-compose.yml) for full configuration.

---

## 📚 Documentation

- **[Getting Started](getting-started/installation.md)** - Installation, quick start, and configuration
- **[Core Concepts](concepts/overview.md)** - Understanding how GraphMem works
- **[Building Agents](agents/guide.md)** - Complete guide to building AI agents
- **[Production](production/architecture.md)** - Deploy at scale with confidence

---

## 🤝 Contributing

We're building the future of AI memory. Join us!

- 🐛 [Report bugs](https://github.com/flancast90/GraphMem-go/issues)
- 💡 [Request features](https://github.com/flancast90/GraphMem-go/issues)
- 🔀 [Submit PRs](https://github.com/flancast90/GraphMem-go/pulls)

---

<div align="center">

**GraphMem-Go** - A Go implementation of GraphMem

*"Give your AI agents the memory they deserve."*

</div>
