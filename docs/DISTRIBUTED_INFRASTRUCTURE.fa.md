# 🚀 راهنمای زیرساخت توزیع‌شده GraphMem

> **معماری کلاس جهانی، با توان عملیاتی بالا و عملکرد فوق‌العاده برای سیستم‌های حافظه هوش مصنوعی در محیط تولید**

این راهنما ساخت زیرساخت توزیع‌شده‌ی GraphMem را پوشش می‌دهد؛ زیرساختی که قادر به پردازش **میلیون‌ها سند**، پاسخ‌گویی به **هزاران پرس‌وجوی همزمان** و مقیاس‌پذیری افقی برای برآورده‌سازی نیازهای سازمانی است.

---

## فهرست مطالب

۱. [نمای کلی معماری](#نمای-کلی-معماری)
۲. [اجزای زیرساخت](#اجزای-زیرساخت)
۳. [لایه صف پیام (Redpanda/Kafka)](#لایه-صف-پیام-redpandakafka)
۴. [معماری استخر کارگرها](#معماری-استخر-کارگرها)
۵. [خوشه تولید بردارهای embedding با GPU](#خوشه-تولید-بردارهای-embedding-با-gpu)
۶. [خط پردازش ناهمگام (Async)](#خط-پردازش-ناهمگام-async)
۷. [لایه ذخیره‌سازی](#لایه-ذخیره‌سازی)
۸. [استقرار در Kubernetes](#استقرار-در-kubernetes)
۹. [داکر کامپوز (توسعه)](#داکر-کامپوز-توسعه)
۱۰. [پایش و مشاهده‌پذیری](#پایش-و-مشاهده‌پذیری)
۱۱. [زیرساخت بنچمارک (سنجش عملکرد)](#زیرساخت-بنچمارک-سنجش-عملکرد)
۱۲. [بهینه‌سازی هزینه](#بهینه‌سازی-هزینه)
۱۳. [چک‌لیست محیط تولید](#چک‌لیست-محیط-تولید)

---

## نمای کلی معماری

### طراحی سیستم در سطح کلان

نمودار زیر نمای سطح‌بالای سیستم توزیع‌شده GraphMem را نشان می‌دهد:

```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                         GRAPHMEM DISTRIBUTED ARCHITECTURE                                │
│                        ══════════════════════════════════════                           │
│                                                                                          │
│  ┌─────────────────┐                                                                    │
│  │   DATA SOURCES  │                                                                    │
│  │                 │                                                                    │
│  │ • REST APIs     │                                                                    │
│  │ • File Uploads  │─────┐                                                              │
│  │ • Webhooks      │     │                                                              │
│  │ • S3 Buckets    │     │                                                              │
│  │ • Databases     │     │                                                              │
│  └─────────────────┘     │                                                              │
│                          ▼                                                              │
│  ┌──────────────────────────────────────────────────────────────────────────────────┐  │
│  │                           INGESTION GATEWAY (FastAPI)                             │  │
│  │                                                                                   │  │
│  │   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │  │
│  │   │ Rate Limiter │  │ Auth/JWT     │  │ Validator    │  │ Router       │        │  │
│  │   └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘        │  │
│  └───────────────────────────────────────────┬──────────────────────────────────────┘  │
│                                              │                                          │
│                                              ▼                                          │
│  ┌──────────────────────────────────────────────────────────────────────────────────┐  │
│  │                        MESSAGE QUEUE (Redpanda/Kafka)                             │  │
│  │                                                                                   │  │
│  │   ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐                 │  │
│  │   │  ingest.raw     │  │  ingest.embed   │  │  ingest.extract │                 │  │
│  │   │  (Documents)    │  │  (For Embed)    │  │  (For LLM)      │                 │  │
│  │   │                 │  │                 │  │                 │                 │  │
│  │   │  Partitions: 32 │  │  Partitions: 16 │  │  Partitions: 16 │                 │  │
│  │   └─────────────────┘  └─────────────────┘  └─────────────────┘                 │  │
│  │                                                                                   │  │
│  │   ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐                 │  │
│  │   │  graph.resolve  │  │  graph.evolve   │  │  query.pending  │                 │  │
│  │   │  (Entity Res)   │  │  (Evolution)    │  │  (Query Queue)  │                 │  │
│  │   └─────────────────┘  └─────────────────┘  └─────────────────┘                 │  │
│  └───────────────────────────────────────────┬──────────────────────────────────────┘  │
│                                              │                                          │
│                                              ▼                                          │
│  ┌──────────────────────────────────────────────────────────────────────────────────┐  │
│  │                              WORKER POOL (K8s)                                    │  │
│  │                                                                                   │  │
│  │   ┌───────────────────────┐  ┌───────────────────────┐  ┌─────────────────────┐ │  │
│  │   │   EMBEDDING WORKERS   │  │  EXTRACTION WORKERS   │  │   GRAPH WORKERS     │ │  │
│  │   │                       │  │                       │  │                     │ │  │
│  │   │  ┌─────┐ ┌─────┐     │  │  ┌─────┐ ┌─────┐     │  │  ┌─────┐ ┌─────┐   │ │  │
│  │   │  │GPU-1│ │GPU-2│ ... │  │  │LLM-1│ │LLM-2│ ... │  │  │GW-1 │ │GW-2 │   │ │  │
│  │   │  └─────┘ └─────┘     │  │  └─────┘ └─────┘     │  │  └─────┘ └─────┘   │ │  │
│  │   │                       │  │                       │  │                     │ │  │
│  │   │  • Batch: 1000 docs   │  │  • Async batch LLM    │  │  • Entity resolve   │ │  │
│  │   │  • GPU: A100/H100     │  │  • Rate limit aware   │  │  • Community detect │ │  │
│  │   │  • Local: E5/BGE      │  │  • Infinite retry     │  │  • Evolution        │ │  │
│  │   │  • Throughput: 10K/s  │  │  • Throughput: 100/s  │  │  • PageRank         │ │  │
│  │   └───────────────────────┘  └───────────────────────┘  └─────────────────────┘ │  │
│  └───────────────────────────────────────────┬──────────────────────────────────────┘  │
│                                              │                                          │
│                                              ▼                                          │
│  ┌──────────────────────────────────────────────────────────────────────────────────┐  │
│  │                              STORAGE LAYER                                        │  │
│  │                                                                                   │  │
│  │   ┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────────┐ │  │
│  │   │    NEO4J CLUSTER    │  │    REDIS CLUSTER    │  │      OBJECT STORAGE     │ │  │
│  │   │                     │  │                     │  │                         │ │  │
│  │   │  ┌───────────────┐  │  │  ┌───────────────┐  │  │  ┌─────────────────────┐│ │  │
│  │   │  │   Primary     │  │  │  │    Master     │  │  │  │    S3 / MinIO       ││ │  │
│  │   │  │   (Write)     │  │  │  │   (Write)     │  │  │  │                     ││ │  │
│  │   │  └───────────────┘  │  │  └───────────────┘  │  │  │  • Raw documents    ││ │  │
│  │   │  ┌───────────────┐  │  │  ┌───────────────┐  │  │  │  • Embeddings cache ││ │  │
│  │   │  │   Replica 1   │  │  │  │   Replica 1   │  │  │  │  • Backups          ││ │  │
│  │   │  │   (Read)      │  │  │  │   (Read)      │  │  │  │  • Snapshots        ││ │  │
│  │   │  └───────────────┘  │  │  └───────────────┘  │  │  └─────────────────────┘│ │  │
│  │   │  ┌───────────────┐  │  │  ┌───────────────┐  │  │                         │ │  │
│  │   │  │   Replica 2   │  │  │  │   Replica 2   │  │  │  ┌─────────────────────┐│ │  │
│  │   │  │   (Read)      │  │  │  │   (Read)      │  │  │  │   Turso (SQLite)    ││ │  │
│  │   │  └───────────────┘  │  │  └───────────────┘  │  │  │   • Vector backup   ││ │  │
│  │   │                     │  │                     │  │  │   • Edge locations  ││ │  │
│  │   │  Entities: 100M+    │  │  Cache: 50GB+       │  │  └─────────────────────┘│ │  │
│  │   │  Relationships: 1B+ │  │  Sessions: 1M+      │  │                         │ │  │
│  │   └─────────────────────┘  └─────────────────────┘  └─────────────────────────┘ │  │
│  └──────────────────────────────────────────────────────────────────────────────────┘  │
│                                                                                          │
│  ┌──────────────────────────────────────────────────────────────────────────────────┐  │
│  │                           OBSERVABILITY LAYER                                     │  │
│  │                                                                                   │  │
│  │   Prometheus  ──▶  Grafana  ──▶  AlertManager  ──▶  PagerDuty/Slack             │  │
│  │   Jaeger      ──▶  Tempo    ──▶  Distributed Tracing                             │  │
│  │   Loki        ──▶  Log Aggregation                                               │  │
│  └──────────────────────────────────────────────────────────────────────────────────┘  │
│                                                                                          │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```

### اهداف عملکردی

| معیار | هدف | روش دستیابی |
| --- | --- | --- |
| **توان عملیاتی ورود داده** | ۱۰٬۰۰۰ سند در ثانیه | بردارهای GPU + LLM ناهمگام |
| **تأخیر پرس‌وجو (p50)** | کمتر از ۱۰۰ میلی‌ثانیه | کش Redis + نسخه‌های خواندنی |
| **تأخیر پرس‌وجو (p99)** | کمتر از ۵۰۰ میلی‌ثانیه | تجمیع اتصالات (Connection Pooling) |
| **کاربران همزمان** | بیش از ۱۰٬۰۰۰ نفر | مقیاس‌پذیری افقی |
| **ظرفیت اسناد** | بیش از ۱۰۰ میلیون سند | شاردینگ Neo4j |
| **دسترس‌پذیری** | ۹۹.۹۹٪ | استقرار چندمنطقه‌ای (Multi-AZ) |

---

## اجزای زیرساخت

### پشته فناوری

| مؤلفه | فناوری | دلیل انتخاب |
| --- | --- | --- |
| **صف پیام** | Redpanda (پیشنهادی) یا Kafka | تأخیر کم، توان عملیاتی بالا، سازگار با API کافکا |
| **تولید بردار (Embedding)** | خوشه GPU (A100/H100) + مدل‌های محلی | ۱۰۰ برابر سریع‌تر از فراخوانی API |
| **استخراج با LLM** | API اوپن‌ای‌آی/آژور + محدودسازی نرخ | بهترین کیفیت استخراج |
| **پایگاه‌داده گرافی** | Neo4j Aura Enterprise | گراف بومی، اسیدیت (ACID)، مقیاس‌پذیری |
| **کش** | Redis Cluster | تأخیر زیر میلی‌ثانیه، انتشار/اشتراک (pub/sub) |
| **ذخیره‌سازی اشیاء** | S3/MinIO | ذخیره‌سازی ارزان و بادوام اسناد |
| **ارکستراسیون** | Kubernetes | مقیاس‌پذیری خودکار، خودترمیمی |
| **پایش** | Prometheus + Grafana + Jaeger | مشاهده‌پذیری کامل |

---

## لایه صف پیام

### چرا Redpanda به جای Kafka؟

| ویژگی | Redpanda | Kafka |
| --- | --- | --- |
| **تأخیر** | حدود ۱۰ms در p99 | حدود ۱۰۰ms در p99 |
| **راه‌اندازی** | یک فایل باینری واحد | ZooKeeper + بروکرها |
| **حافظه** | ۱۰ برابر کمتر | سربار JVM |
| **سازگاری** | ۱۰۰٪ API کافکا | بومی |

### پیکربندی تاپیک‌ها

```yaml
# redpanda-topics.yaml
topics:
  - name: graphmem.ingest.raw
    partitions: 32
    replication_factor: 3
    config:
      retention.ms: 604800000  # 7 days
      max.message.bytes: 10485760  # 10MB
      
  - name: graphmem.ingest.embed
    partitions: 16
    replication_factor: 3
    config:
      retention.ms: 86400000  # 1 day
      
  - name: graphmem.ingest.extract
    partitions: 16
    replication_factor: 3
    config:
      retention.ms: 86400000  # 1 day
      
  - name: graphmem.graph.resolve
    partitions: 8
    replication_factor: 3
    
  - name: graphmem.graph.evolve
    partitions: 4
    replication_factor: 3
    
  - name: graphmem.query.pending
    partitions: 32
    replication_factor: 3
    config:
      retention.ms: 3600000  # 1 hour
      
  - name: graphmem.dlq  # Dead letter queue
    partitions: 4
    replication_factor: 3
    config:
      retention.ms: 2592000000  # 30 days
```

### پیاده‌سازی تولیدکننده (Producer)

```python
# graphmem/distributed/producer.py
"""
High-performance message producer for GraphMem distributed ingestion.
"""

import json
import asyncio
from typing import List, Dict, Any, Optional
from dataclasses import dataclass
from confluent_kafka import Producer
from confluent_kafka.admin import AdminClient, NewTopic
import hashlib

@dataclass
class ProducerConfig:
    bootstrap_servers: str = "localhost:9092"
    batch_size: int = 100000  # 100KB batches
    linger_ms: int = 10  # Wait up to 10ms to batch
    compression: str = "lz4"
    acks: str = "all"  # Wait for all replicas
    retries: int = 10
    
class GraphMemProducer:
    """
    High-throughput producer for distributing GraphMem workloads.
    
    Features:
    - Automatic batching for efficiency
    - Partitioning by tenant/memory_id for locality
    - Compression for bandwidth savings
    - At-least-once delivery guarantees
    """
    
    TOPICS = {
        "raw": "graphmem.ingest.raw",
        "embed": "graphmem.ingest.embed",
        "extract": "graphmem.ingest.extract",
        "resolve": "graphmem.graph.resolve",
        "evolve": "graphmem.graph.evolve",
        "query": "graphmem.query.pending",
        "dlq": "graphmem.dlq",
    }
    
    def __init__(self, config: ProducerConfig = None):
        self.config = config or ProducerConfig()
        
        self.producer = Producer({
            'bootstrap.servers': self.config.bootstrap_servers,
            'batch.size': self.config.batch_size,
            'linger.ms': self.config.linger_ms,
            'compression.type': self.config.compression,
            'acks': self.config.acks,
            'retries': self.config.retries,
            'enable.idempotence': True,  # Exactly-once semantics
        })
        
        self._pending = 0
        self._errors = []
    
    def _delivery_callback(self, err, msg):
        """Callback for message delivery confirmation."""
        self._pending -= 1
        if err:
            self._errors.append({
                'error': str(err),
                'topic': msg.topic(),
                'partition': msg.partition(),
            })
    
    def _partition_key(self, tenant_id: str, memory_id: str) -> bytes:
        """Generate partition key for locality."""
        key = f"{tenant_id}:{memory_id}"
        return hashlib.md5(key.encode()).digest()
    
    def send_document(
        self,
        document: Dict[str, Any],
        tenant_id: str,
        memory_id: str,
        priority: int = 5,
    ):
        """
        Send a document for ingestion.
        
        Args:
            document: Document with 'id', 'content', and optional metadata
            tenant_id: Tenant identifier for isolation
            memory_id: Memory instance ID
            priority: 1-10, higher = process first
        """
        message = {
            'type': 'document',
            'tenant_id': tenant_id,
            'memory_id': memory_id,
            'priority': priority,
            'payload': document,
        }
        
        self.producer.produce(
            topic=self.TOPICS['raw'],
            key=self._partition_key(tenant_id, memory_id),
            value=json.dumps(message).encode('utf-8'),
            callback=self._delivery_callback,
        )
        self._pending += 1
        
        # Periodic poll to handle callbacks
        if self._pending % 1000 == 0:
            self.producer.poll(0)
    
    def send_batch(
        self,
        documents: List[Dict[str, Any]],
        tenant_id: str,
        memory_id: str,
    ):
        """Send a batch of documents efficiently."""
        for doc in documents:
            self.send_document(doc, tenant_id, memory_id)
        
        # Flush at end of batch
        self.producer.poll(0)
    
    def send_embedding_request(
        self,
        text_batch: List[str],
        doc_ids: List[str],
        tenant_id: str,
    ):
        """Send texts for embedding generation."""
        message = {
            'type': 'embed_batch',
            'tenant_id': tenant_id,
            'texts': text_batch,
            'doc_ids': doc_ids,
        }
        
        self.producer.produce(
            topic=self.TOPICS['embed'],
            key=tenant_id.encode(),
            value=json.dumps(message).encode('utf-8'),
            callback=self._delivery_callback,
        )
        self._pending += 1
    
    def send_extraction_request(
        self,
        chunk: str,
        chunk_id: str,
        metadata: Dict[str, Any],
        tenant_id: str,
        memory_id: str,
    ):
        """Send a chunk for LLM extraction."""
        message = {
            'type': 'extract',
            'tenant_id': tenant_id,
            'memory_id': memory_id,
            'chunk_id': chunk_id,
            'chunk': chunk,
            'metadata': metadata,
        }
        
        self.producer.produce(
            topic=self.TOPICS['extract'],
            key=self._partition_key(tenant_id, memory_id),
            value=json.dumps(message).encode('utf-8'),
            callback=self._delivery_callback,
        )
        self._pending += 1
    
    def send_query(
        self,
        query: str,
        tenant_id: str,
        memory_id: str,
        callback_url: Optional[str] = None,
    ) -> str:
        """Send a query for async processing."""
        import uuid
        query_id = str(uuid.uuid4())
        
        message = {
            'type': 'query',
            'query_id': query_id,
            'tenant_id': tenant_id,
            'memory_id': memory_id,
            'query': query,
            'callback_url': callback_url,
        }
        
        self.producer.produce(
            topic=self.TOPICS['query'],
            key=self._partition_key(tenant_id, memory_id),
            value=json.dumps(message).encode('utf-8'),
            callback=self._delivery_callback,
        )
        self._pending += 1
        
        return query_id
    
    def flush(self, timeout: float = 30.0):
        """Flush all pending messages."""
        remaining = self.producer.flush(timeout)
        if remaining > 0:
            raise RuntimeError(f"{remaining} messages failed to send")
        
        if self._errors:
            errors = self._errors.copy()
            self._errors.clear()
            raise RuntimeError(f"Delivery errors: {errors}")
    
    def close(self):
        """Close producer and flush pending messages."""
        self.flush()
        self.producer = None


# Convenience function
def create_producer(bootstrap_servers: str = "localhost:9092") -> GraphMemProducer:
    """Create a configured producer."""
    return GraphMemProducer(ProducerConfig(bootstrap_servers=bootstrap_servers))
```

---

## معماری استخر کارگرها

### پیاده‌سازی مصرف‌کننده/کارگر

```python
# graphmem/distributed/worker.py
"""
Distributed worker for processing GraphMem tasks.

Supports:
- Embedding generation (GPU)
- LLM extraction (API)
- Graph operations (CPU)
- Query processing (CPU)
"""

import json
import asyncio
import logging
from typing import Dict, Any, Callable, Optional
from dataclasses import dataclass
from concurrent.futures import ThreadPoolExecutor
from confluent_kafka import Consumer, KafkaError, KafkaException
import signal
import sys

logger = logging.getLogger(__name__)

@dataclass
class WorkerConfig:
    bootstrap_servers: str = "localhost:9092"
    group_id: str = "graphmem-workers"
    worker_type: str = "general"  # general, embed, extract, graph, query
    max_poll_records: int = 100
    poll_timeout: float = 1.0
    commit_interval: int = 100
    max_workers: int = 10

class DistributedWorker:
    """
    Kafka consumer worker for distributed GraphMem processing.
    
    Worker Types:
    - embed: GPU-accelerated embedding generation
    - extract: LLM-based knowledge extraction
    - graph: Entity resolution, community detection, evolution
    - query: Query processing and response generation
    - general: Can handle any task
    """
    
    TOPIC_MAP = {
        "embed": ["graphmem.ingest.embed"],
        "extract": ["graphmem.ingest.extract"],
        "graph": ["graphmem.graph.resolve", "graphmem.graph.evolve"],
        "query": ["graphmem.query.pending"],
        "general": [
            "graphmem.ingest.raw",
            "graphmem.ingest.embed",
            "graphmem.ingest.extract",
            "graphmem.graph.resolve",
            "graphmem.query.pending",
        ],
    }
    
    def __init__(
        self,
        config: WorkerConfig,
        graphmem_factory: Callable[[], Any],  # Factory to create GraphMem instances
    ):
        self.config = config
        self.graphmem_factory = graphmem_factory
        self._running = False
        self._processed = 0
        
        # Initialize consumer
        self.consumer = Consumer({
            'bootstrap.servers': config.bootstrap_servers,
            'group.id': f"{config.group_id}-{config.worker_type}",
            'auto.offset.reset': 'earliest',
            'enable.auto.commit': False,  # Manual commit for exactly-once
            'max.poll.records': config.max_poll_records,
        })
        
        # Subscribe to relevant topics
        topics = self.TOPIC_MAP.get(config.worker_type, self.TOPIC_MAP['general'])
        self.consumer.subscribe(topics)
        logger.info(f"Worker subscribed to: {topics}")
        
        # Thread pool for parallel processing
        self.executor = ThreadPoolExecutor(max_workers=config.max_workers)
        
        # GraphMem instances per tenant (lazy init)
        self._graphmem_instances: Dict[str, Any] = {}
        
        # Handlers for different message types
        self._handlers = {
            'document': self._handle_document,
            'embed_batch': self._handle_embed_batch,
            'extract': self._handle_extract,
            'query': self._handle_query,
            'resolve': self._handle_resolve,
            'evolve': self._handle_evolve,
        }
    
    def _get_graphmem(self, tenant_id: str, memory_id: str) -> Any:
        """Get or create GraphMem instance for tenant."""
        key = f"{tenant_id}:{memory_id}"
        if key not in self._graphmem_instances:
            self._graphmem_instances[key] = self.graphmem_factory(tenant_id, memory_id)
        return self._graphmem_instances[key]
    
    def _handle_document(self, message: Dict[str, Any]):
        """Handle raw document ingestion."""
        tenant_id = message['tenant_id']
        memory_id = message['memory_id']
        document = message['payload']
        
        gm = self._get_graphmem(tenant_id, memory_id)
        gm.ingest(
            content=document.get('content', document.get('text', '')),
            metadata={k: v for k, v in document.items() if k not in ('content', 'text')},
        )
        
        logger.debug(f"Ingested document for {tenant_id}/{memory_id}")
    
    def _handle_embed_batch(self, message: Dict[str, Any]):
        """Handle batch embedding generation."""
        tenant_id = message['tenant_id']
        texts = message['texts']
        doc_ids = message['doc_ids']
        
        # Use local embedding model for speed
        from sentence_transformers import SentenceTransformer
        model = SentenceTransformer('BAAI/bge-large-en-v1.5')
        
        embeddings = model.encode(texts, batch_size=256, show_progress_bar=False)
        
        # Store embeddings (could publish to another topic or store directly)
        logger.info(f"Generated {len(embeddings)} embeddings for {tenant_id}")
        
        return {'embeddings': embeddings.tolist(), 'doc_ids': doc_ids}
    
    def _handle_extract(self, message: Dict[str, Any]):
        """Handle LLM extraction."""
        tenant_id = message['tenant_id']
        memory_id = message['memory_id']
        chunk = message['chunk']
        metadata = message['metadata']
        
        gm = self._get_graphmem(tenant_id, memory_id)
        
        # Extract from chunk
        nodes, edges = gm._knowledge_graph._extract_from_chunk(
            chunk=chunk,
            metadata=metadata,
            memory_id=memory_id,
            user_id=tenant_id,
        )
        
        logger.debug(f"Extracted {len(nodes)} nodes, {len(edges)} edges")
        
        return {'nodes': len(nodes), 'edges': len(edges)}
    
    def _handle_query(self, message: Dict[str, Any]):
        """Handle query processing."""
        tenant_id = message['tenant_id']
        memory_id = message['memory_id']
        query = message['query']
        query_id = message['query_id']
        callback_url = message.get('callback_url')
        
        gm = self._get_graphmem(tenant_id, memory_id)
        response = gm.query(query)
        
        result = {
            'query_id': query_id,
            'answer': response.answer,
            'confidence': response.confidence,
            'latency_ms': response.latency_ms,
        }
        
        # Send callback if URL provided
        if callback_url:
            import requests
            requests.post(callback_url, json=result)
        
        return result
    
    def _handle_resolve(self, message: Dict[str, Any]):
        """Handle entity resolution."""
        tenant_id = message['tenant_id']
        memory_id = message['memory_id']
        
        gm = self._get_graphmem(tenant_id, memory_id)
        # Trigger resolution on the memory
        # This would be part of the evolution process
        
        logger.debug(f"Entity resolution for {tenant_id}/{memory_id}")
    
    def _handle_evolve(self, message: Dict[str, Any]):
        """Handle memory evolution."""
        tenant_id = message['tenant_id']
        memory_id = message['memory_id']
        
        gm = self._get_graphmem(tenant_id, memory_id)
        events = gm.evolve(force=True)
        
        logger.info(f"Evolution completed: {len(events)} events for {tenant_id}/{memory_id}")
        
        return {'events': len(events)}
    
    def run(self):
        """Main worker loop."""
        self._running = True
        
        # Handle graceful shutdown
        def signal_handler(sig, frame):
            logger.info("Shutdown signal received")
            self._running = False
        
        signal.signal(signal.SIGINT, signal_handler)
        signal.signal(signal.SIGTERM, signal_handler)
        
        logger.info(f"Worker starting: type={self.config.worker_type}")
        
        pending_commits = 0
        
        try:
            while self._running:
                msg = self.consumer.poll(self.config.poll_timeout)
                
                if msg is None:
                    continue
                
                if msg.error():
                    if msg.error().code() == KafkaError._PARTITION_EOF:
                        continue
                    else:
                        raise KafkaException(msg.error())
                
                try:
                    # Parse message
                    value = json.loads(msg.value().decode('utf-8'))
                    msg_type = value.get('type', 'unknown')
                    
                    # Find handler
                    handler = self._handlers.get(msg_type)
                    if handler:
                        # Process in thread pool
                        future = self.executor.submit(handler, value)
                        # For now, wait for result (could be made fully async)
                        result = future.result(timeout=300)
                        
                        self._processed += 1
                        pending_commits += 1
                        
                        # Periodic commit
                        if pending_commits >= self.config.commit_interval:
                            self.consumer.commit()
                            pending_commits = 0
                    else:
                        logger.warning(f"Unknown message type: {msg_type}")
                        
                except Exception as e:
                    logger.error(f"Error processing message: {e}")
                    # Could send to DLQ here
        
        finally:
            # Final commit
            if pending_commits > 0:
                self.consumer.commit()
            
            self.consumer.close()
            self.executor.shutdown(wait=True)
            
            logger.info(f"Worker stopped. Processed {self._processed} messages.")


def run_worker(
    worker_type: str = "general",
    bootstrap_servers: str = "localhost:9092",
    graphmem_config: Dict[str, Any] = None,
):
    """
    Convenience function to run a worker.
    
    Args:
        worker_type: Type of worker (embed, extract, graph, query, general)
        bootstrap_servers: Kafka/Redpanda bootstrap servers
        graphmem_config: Configuration for GraphMem instances
    """
    from graphmem import GraphMem, MemoryConfig
    
    def graphmem_factory(tenant_id: str, memory_id: str):
        """Create GraphMem instance for tenant."""
        config = MemoryConfig(
            **(graphmem_config or {}),
            user_id=tenant_id,
            memory_id=memory_id,
        )
        return GraphMem(config)
    
    worker_config = WorkerConfig(
        bootstrap_servers=bootstrap_servers,
        worker_type=worker_type,
    )
    
    worker = DistributedWorker(worker_config, graphmem_factory)
    worker.run()


if __name__ == "__main__":
    import argparse
    
    parser = argparse.ArgumentParser()
    parser.add_argument("--type", default="general", choices=["embed", "extract", "graph", "query", "general"])
    parser.add_argument("--servers", default="localhost:9092")
    args = parser.parse_args()
    
    run_worker(worker_type=args.type, bootstrap_servers=args.servers)
```

---

## خوشه تولید بردار با GPU

### سرویس تولید بردار با عملکرد بالا

```python
# graphmem/distributed/embedding_service.py
"""
GPU-accelerated embedding service for high-throughput ingestion.

Supports:
- Batch processing (1000+ docs at once)
- Multiple GPU workers
- Local models (no API costs)
- Caching layer
"""

import torch
import numpy as np
from typing import List, Optional
from dataclasses import dataclass
import logging
import redis
import hashlib
from concurrent.futures import ThreadPoolExecutor

logger = logging.getLogger(__name__)

@dataclass
class EmbeddingConfig:
    model_name: str = "BAAI/bge-large-en-v1.5"
    batch_size: int = 256
    max_length: int = 512
    device: str = "auto"  # auto, cuda, cpu
    num_workers: int = 4
    redis_url: Optional[str] = None
    cache_ttl: int = 86400 * 7  # 7 days

class GPUEmbeddingService:
    """
    High-performance embedding service using local GPU models.
    
    Performance:
    - Single A100: ~5,000 embeddings/sec
    - Single H100: ~10,000 embeddings/sec
    - With batching: 100x faster than API calls
    
    Features:
    - Automatic batching
    - Redis caching
    - Multi-GPU support
    - Graceful degradation to CPU
    """
    
    def __init__(self, config: EmbeddingConfig = None):
        self.config = config or EmbeddingConfig()
        
        # Detect device
        if self.config.device == "auto":
            if torch.cuda.is_available():
                self.device = "cuda"
                logger.info(f"Using GPU: {torch.cuda.get_device_name(0)}")
            else:
                self.device = "cpu"
                logger.info("GPU not available, using CPU")
        else:
            self.device = self.config.device
        
        # Load model
        from sentence_transformers import SentenceTransformer
        self.model = SentenceTransformer(
            self.config.model_name,
            device=self.device,
        )
        
        # Enable multi-GPU if available
        if self.device == "cuda" and torch.cuda.device_count() > 1:
            logger.info(f"Using {torch.cuda.device_count()} GPUs")
            # DataParallel for multi-GPU
            self.model = torch.nn.DataParallel(self.model)
        
        # Redis cache
        self.cache = None
        if self.config.redis_url:
            self.cache = redis.from_url(self.config.redis_url)
            logger.info("Redis caching enabled")
        
        # Thread pool for async operations
        self.executor = ThreadPoolExecutor(max_workers=self.config.num_workers)
    
    def _cache_key(self, text: str) -> str:
        """Generate cache key for text."""
        return f"embed:{hashlib.md5(text.encode()).hexdigest()}"
    
    def _get_cached(self, texts: List[str]) -> tuple:
        """Get cached embeddings, return (cached, uncached_indices)."""
        if not self.cache:
            return {}, list(range(len(texts)))
        
        cached = {}
        uncached = []
        
        # Batch get from Redis
        keys = [self._cache_key(t) for t in texts]
        results = self.cache.mget(keys)
        
        for i, (text, result) in enumerate(zip(texts, results)):
            if result:
                cached[i] = np.frombuffer(result, dtype=np.float32)
            else:
                uncached.append(i)
        
        return cached, uncached
    
    def _set_cached(self, texts: List[str], embeddings: np.ndarray, indices: List[int]):
        """Cache embeddings."""
        if not self.cache:
            return
        
        pipe = self.cache.pipeline()
        for i, idx in enumerate(indices):
            key = self._cache_key(texts[idx])
            pipe.setex(key, self.config.cache_ttl, embeddings[i].tobytes())
        pipe.execute()
    
    def embed(self, texts: List[str]) -> np.ndarray:
        """
        Generate embeddings for a list of texts.
        
        Args:
            texts: List of texts to embed
            
        Returns:
            numpy array of shape (len(texts), embedding_dim)
        """
        if not texts:
            return np.array([])
        
        # Check cache
        cached, uncached_indices = self._get_cached(texts)
        
        if not uncached_indices:
            # All cached!
            embeddings = np.zeros((len(texts), len(list(cached.values())[0])))
            for i, emb in cached.items():
                embeddings[i] = emb
            return embeddings
        
        # Generate embeddings for uncached texts
        uncached_texts = [texts[i] for i in uncached_indices]
        
        with torch.no_grad():
            new_embeddings = self.model.encode(
                uncached_texts,
                batch_size=self.config.batch_size,
                show_progress_bar=False,
                convert_to_numpy=True,
                normalize_embeddings=True,  # L2 normalize for cosine similarity
            )
        
        # Cache new embeddings
        self._set_cached(texts, new_embeddings, uncached_indices)
        
        # Combine cached and new
        embedding_dim = new_embeddings.shape[1]
        all_embeddings = np.zeros((len(texts), embedding_dim))
        
        for i, emb in cached.items():
            all_embeddings[i] = emb
        
        for i, idx in enumerate(uncached_indices):
            all_embeddings[idx] = new_embeddings[i]
        
        return all_embeddings
    
    def embed_single(self, text: str) -> np.ndarray:
        """Embed a single text."""
        return self.embed([text])[0]
    
    async def embed_async(self, texts: List[str]) -> np.ndarray:
        """Async embedding for non-blocking operations."""
        import asyncio
        loop = asyncio.get_event_loop()
        return await loop.run_in_executor(self.executor, self.embed, texts)
    
    def benchmark(self, num_texts: int = 10000) -> dict:
        """Benchmark embedding performance."""
        import time
        
        # Generate test texts
        texts = [f"This is test text number {i} for benchmarking." for i in range(num_texts)]
        
        # Warm up
        self.embed(texts[:100])
        
        # Benchmark
        start = time.time()
        embeddings = self.embed(texts)
        elapsed = time.time() - start
        
        return {
            "num_texts": num_texts,
            "elapsed_seconds": elapsed,
            "throughput": num_texts / elapsed,
            "embedding_dim": embeddings.shape[1],
            "device": self.device,
            "cache_hit_rate": 0 if not self.cache else "enabled",
        }


# FastAPI service for remote embedding
def create_embedding_api():
    """Create FastAPI app for embedding service."""
    from fastapi import FastAPI
    from pydantic import BaseModel
    
    app = FastAPI(title="GraphMem Embedding Service")
    service = GPUEmbeddingService()
    
    class EmbedRequest(BaseModel):
        texts: List[str]
    
    class EmbedResponse(BaseModel):
        embeddings: List[List[float]]
    
    @app.post("/embed", response_model=EmbedResponse)
    async def embed(request: EmbedRequest):
        embeddings = await service.embed_async(request.texts)
        return EmbedResponse(embeddings=embeddings.tolist())
    
    @app.get("/health")
    async def health():
        return {"status": "healthy", "device": service.device}
    
    @app.get("/benchmark")
    async def benchmark():
        return service.benchmark(num_texts=1000)
    
    return app


if __name__ == "__main__":
    # Run as standalone service
    import uvicorn
    app = create_embedding_api()
    uvicorn.run(app, host="0.0.0.0", port=8001)
```

---

## لایه ذخیره‌سازی

### پیکربندی خوشه Neo4j

```yaml
# neo4j-cluster.yaml
# Deploy via Neo4j Helm chart or Aura

# For self-hosted Neo4j cluster:
neo4j:
  core:
    numberOfServers: 3
    resources:
      requests:
        cpu: "4"
        memory: "16Gi"
      limits:
        cpu: "8"
        memory: "32Gi"
    
    persistentVolume:
      size: 500Gi
      storageClass: "fast-ssd"
    
    config:
      # Performance tuning
      dbms.memory.heap.initial_size: "8G"
      dbms.memory.heap.max_size: "16G"
      dbms.memory.pagecache.size: "8G"
      
      # Connection pooling
      dbms.connector.bolt.connection_keep_alive: "true"
      dbms.connector.bolt.connection_keep_alive_for_requests: "ALL"
      
      # Query optimization
      cypher.min_replan_interval: "10s"
      dbms.query_cache_size: "1000"
  
  readReplica:
    numberOfServers: 2
    resources:
      requests:
        cpu: "2"
        memory: "8Gi"
```

### پیکربندی خوشه Redis

```yaml
# redis-cluster.yaml
# Deploy via Redis Helm chart

redis:
  cluster:
    enabled: true
    slotsMigrationTimeout: 15000
    nodes: 6  # 3 masters + 3 replicas
  
  master:
    resources:
      requests:
        cpu: "2"
        memory: "8Gi"
      limits:
        cpu: "4"
        memory: "16Gi"
    
    persistence:
      size: 50Gi
  
  replica:
    replicaCount: 1
    resources:
      requests:
        cpu: "1"
        memory: "4Gi"
  
  # Performance tuning
  commonConfiguration: |-
    maxmemory 12gb
    maxmemory-policy allkeys-lru
    tcp-keepalive 300
    timeout 0
    tcp-backlog 511
```

---

## استقرار در Kubernetes

### مانیفست‌های کامل Kubernetes

```yaml
# k8s/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: graphmem

---
# k8s/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: graphmem-config
  namespace: graphmem
data:
  KAFKA_BOOTSTRAP_SERVERS: "redpanda.graphmem.svc:9092"
  REDIS_URL: "redis://redis-cluster.graphmem.svc:6379"
  NEO4J_URI: "neo4j://neo4j-cluster.graphmem.svc:7687"
  LOG_LEVEL: "INFO"
  WORKER_MAX_WORKERS: "10"

---
# k8s/secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: graphmem-secrets
  namespace: graphmem
type: Opaque
stringData:
  OPENAI_API_KEY: "sk-..."
  NEO4J_PASSWORD: "..."
  REDIS_PASSWORD: "..."

---
# k8s/gateway-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: graphmem-gateway
  namespace: graphmem
spec:
  replicas: 3
  selector:
    matchLabels:
      app: graphmem-gateway
  template:
    metadata:
      labels:
        app: graphmem-gateway
    spec:
      containers:
      - name: gateway
        image: graphmem/gateway:latest
        ports:
        - containerPort: 8000
        envFrom:
        - configMapRef:
            name: graphmem-config
        - secretRef:
            name: graphmem-secrets
        resources:
          requests:
            cpu: "500m"
            memory: "1Gi"
          limits:
            cpu: "2"
            memory: "4Gi"
        livenessProbe:
          httpGet:
            path: /health
            port: 8000
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8000
          initialDelaySeconds: 5
          periodSeconds: 5

---
# k8s/gateway-service.yaml
apiVersion: v1
kind: Service
metadata:
  name: graphmem-gateway
  namespace: graphmem
spec:
  selector:
    app: graphmem-gateway
  ports:
  - port: 80
    targetPort: 8000
  type: LoadBalancer

---
# k8s/gateway-hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: graphmem-gateway-hpa
  namespace: graphmem
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: graphmem-gateway
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80

---
# k8s/embedding-worker-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: graphmem-embed-worker
  namespace: graphmem
spec:
  replicas: 2  # Scale based on GPU availability
  selector:
    matchLabels:
      app: graphmem-embed-worker
  template:
    metadata:
      labels:
        app: graphmem-embed-worker
    spec:
      containers:
      - name: worker
        image: graphmem/worker:latest
        args: ["--type", "embed"]
        envFrom:
        - configMapRef:
            name: graphmem-config
        - secretRef:
            name: graphmem-secrets
        resources:
          requests:
            cpu: "4"
            memory: "16Gi"
            nvidia.com/gpu: "1"
          limits:
            cpu: "8"
            memory: "32Gi"
            nvidia.com/gpu: "1"
      nodeSelector:
        gpu: "true"
      tolerations:
      - key: "nvidia.com/gpu"
        operator: "Exists"
        effect: "NoSchedule"

---
# k8s/extract-worker-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: graphmem-extract-worker
  namespace: graphmem
spec:
  replicas: 10
  selector:
    matchLabels:
      app: graphmem-extract-worker
  template:
    metadata:
      labels:
        app: graphmem-extract-worker
    spec:
      containers:
      - name: worker
        image: graphmem/worker:latest
        args: ["--type", "extract"]
        envFrom:
        - configMapRef:
            name: graphmem-config
        - secretRef:
            name: graphmem-secrets
        resources:
          requests:
            cpu: "500m"
            memory: "1Gi"
          limits:
            cpu: "2"
            memory: "4Gi"

---
# k8s/extract-worker-hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: graphmem-extract-worker-hpa
  namespace: graphmem
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: graphmem-extract-worker
  minReplicas: 5
  maxReplicas: 50
  metrics:
  - type: External
    external:
      metric:
        name: kafka_consumer_lag
        selector:
          matchLabels:
            topic: graphmem.ingest.extract
      target:
        type: AverageValue
        averageValue: "1000"  # Scale up if lag > 1000

---
# k8s/query-worker-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: graphmem-query-worker
  namespace: graphmem
spec:
  replicas: 5
  selector:
    matchLabels:
      app: graphmem-query-worker
  template:
    metadata:
      labels:
        app: graphmem-query-worker
    spec:
      containers:
      - name: worker
        image: graphmem/worker:latest
        args: ["--type", "query"]
        envFrom:
        - configMapRef:
            name: graphmem-config
        - secretRef:
            name: graphmem-secrets
        resources:
          requests:
            cpu: "1"
            memory: "2Gi"
          limits:
            cpu: "4"
            memory: "8Gi"
```

---

## داکر کامپوز (محیط توسعه)

```yaml
# docker-compose.yml
version: '3.8'

services:
  # Message Queue
  redpanda:
    image: vectorized/redpanda:latest
    command:
      - redpanda
      - start
      - --smp 1
      - --memory 1G
      - --reserve-memory 0M
      - --overprovisioned
      - --node-id 0
      - --kafka-addr PLAINTEXT://0.0.0.0:9092
    ports:
      - "9092:9092"
      - "8081:8081"  # Schema registry
      - "8082:8082"  # REST proxy
    volumes:
      - redpanda-data:/var/lib/redpanda/data

  # Redis Cache
  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes --maxmemory 2gb --maxmemory-policy allkeys-lru
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data

  # Neo4j Graph Database
  neo4j:
    image: neo4j:5-enterprise
    ports:
      - "7474:7474"  # HTTP
      - "7687:7687"  # Bolt
    environment:
      NEO4J_AUTH: neo4j/graphmem123
      NEO4J_ACCEPT_LICENSE_AGREEMENT: "yes"
      NEO4J_dbms_memory_heap_initial__size: "2G"
      NEO4J_dbms_memory_heap_max__size: "4G"
      NEO4J_dbms_memory_pagecache_size: "2G"
    volumes:
      - neo4j-data:/data

  # GraphMem Gateway
  gateway:
    build:
      context: .
      dockerfile: Dockerfile.gateway
    ports:
      - "8000:8000"
    environment:
      KAFKA_BOOTSTRAP_SERVERS: redpanda:9092
      REDIS_URL: redis://redis:6379
      NEO4J_URI: neo4j://neo4j:7687
      NEO4J_USER: neo4j
      NEO4J_PASSWORD: graphmem123
      OPENAI_API_KEY: ${OPENAI_API_KEY}
    depends_on:
      - redpanda
      - redis
      - neo4j

  # Embedding Worker (GPU)
  embed-worker:
    build:
      context: .
      dockerfile: Dockerfile.worker
    command: ["--type", "embed"]
    environment:
      KAFKA_BOOTSTRAP_SERVERS: redpanda:9092
      REDIS_URL: redis://redis:6379
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]
    depends_on:
      - redpanda
      - redis

  # Extraction Workers (scale these)
  extract-worker:
    build:
      context: .
      dockerfile: Dockerfile.worker
    command: ["--type", "extract"]
    environment:
      KAFKA_BOOTSTRAP_SERVERS: redpanda:9092
      REDIS_URL: redis://redis:6379
      NEO4J_URI: neo4j://neo4j:7687
      NEO4J_USER: neo4j
      NEO4J_PASSWORD: graphmem123
      OPENAI_API_KEY: ${OPENAI_API_KEY}
    depends_on:
      - redpanda
      - redis
      - neo4j
    deploy:
      replicas: 5

  # Query Workers
  query-worker:
    build:
      context: .
      dockerfile: Dockerfile.worker
    command: ["--type", "query"]
    environment:
      KAFKA_BOOTSTRAP_SERVERS: redpanda:9092
      REDIS_URL: redis://redis:6379
      NEO4J_URI: neo4j://neo4j:7687
      NEO4J_USER: neo4j
      NEO4J_PASSWORD: graphmem123
      OPENAI_API_KEY: ${OPENAI_API_KEY}
    depends_on:
      - redpanda
      - redis
      - neo4j
    deploy:
      replicas: 3

  # Monitoring
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus-data:/prometheus

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      GF_SECURITY_ADMIN_PASSWORD: admin
    volumes:
      - grafana-data:/var/lib/grafana
      - ./grafana/dashboards:/etc/grafana/provisioning/dashboards
      - ./grafana/datasources:/etc/grafana/provisioning/datasources

volumes:
  redpanda-data:
  redis-data:
  neo4j-data:
  prometheus-data:
  grafana-data:
```

### داکرفایل‌ها

```dockerfile
# Dockerfile.gateway
FROM python:3.11-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

EXPOSE 8000

CMD ["uvicorn", "gateway:app", "--host", "0.0.0.0", "--port", "8000", "--workers", "4"]
```

```dockerfile
# Dockerfile.worker
FROM python:3.11-slim

# For GPU support, use nvidia base image:
# FROM nvidia/cuda:12.1-runtime-ubuntu22.04

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Install sentence-transformers for local embeddings
RUN pip install sentence-transformers torch

COPY . .

ENTRYPOINT ["python", "-m", "graphmem.distributed.worker"]
```

---

## پایش و مشاهده‌پذیری

### پیکربندی Prometheus

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093

scrape_configs:
  - job_name: 'graphmem-gateway'
    static_configs:
      - targets: ['gateway:8000']
    metrics_path: /metrics

  - job_name: 'graphmem-workers'
    static_configs:
      - targets: 
        - 'embed-worker:8001'
        - 'extract-worker:8001'
        - 'query-worker:8001'

  - job_name: 'redpanda'
    static_configs:
      - targets: ['redpanda:9644']

  - job_name: 'redis'
    static_configs:
      - targets: ['redis:6379']

  - job_name: 'neo4j'
    static_configs:
      - targets: ['neo4j:2004']
```

### معیارهای کلیدی برای پایش

```python
# graphmem/distributed/metrics.py
"""
Prometheus metrics for GraphMem distributed system.
"""

from prometheus_client import Counter, Histogram, Gauge, Info

# Ingestion metrics
DOCS_INGESTED = Counter(
    'graphmem_docs_ingested_total',
    'Total documents ingested',
    ['tenant_id', 'status']
)

INGESTION_LATENCY = Histogram(
    'graphmem_ingestion_latency_seconds',
    'Document ingestion latency',
    ['tenant_id'],
    buckets=[0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0, 60.0]
)

# Extraction metrics
ENTITIES_EXTRACTED = Counter(
    'graphmem_entities_extracted_total',
    'Total entities extracted',
    ['tenant_id', 'entity_type']
)

RELATIONSHIPS_EXTRACTED = Counter(
    'graphmem_relationships_extracted_total',
    'Total relationships extracted',
    ['tenant_id']
)

EXTRACTION_LATENCY = Histogram(
    'graphmem_extraction_latency_seconds',
    'LLM extraction latency per chunk',
    ['tenant_id'],
    buckets=[0.5, 1.0, 2.0, 5.0, 10.0, 30.0]
)

# Query metrics
QUERIES_TOTAL = Counter(
    'graphmem_queries_total',
    'Total queries processed',
    ['tenant_id', 'status']
)

QUERY_LATENCY = Histogram(
    'graphmem_query_latency_seconds',
    'Query processing latency',
    ['tenant_id'],
    buckets=[0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0]
)

QUERY_CONFIDENCE = Histogram(
    'graphmem_query_confidence',
    'Query answer confidence distribution',
    ['tenant_id'],
    buckets=[0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0]
)

# Cache metrics
CACHE_HITS = Counter(
    'graphmem_cache_hits_total',
    'Cache hit count',
    ['cache_type']  # query, embedding, state
)

CACHE_MISSES = Counter(
    'graphmem_cache_misses_total',
    'Cache miss count',
    ['cache_type']
)

# Worker metrics
WORKER_ACTIVE = Gauge(
    'graphmem_workers_active',
    'Number of active workers',
    ['worker_type']
)

WORKER_QUEUE_LAG = Gauge(
    'graphmem_worker_queue_lag',
    'Consumer lag (messages behind)',
    ['worker_type', 'topic']
)

# Storage metrics
NEO4J_NODES = Gauge(
    'graphmem_neo4j_nodes_total',
    'Total nodes in Neo4j',
    ['tenant_id']
)

NEO4J_RELATIONSHIPS = Gauge(
    'graphmem_neo4j_relationships_total',
    'Total relationships in Neo4j',
    ['tenant_id']
)

# Evolution metrics
EVOLUTION_EVENTS = Counter(
    'graphmem_evolution_events_total',
    'Evolution events',
    ['tenant_id', 'event_type']  # consolidation, decay, synthesis
)
```

### داشبورد Grafana

```json
{
  "dashboard": {
    "title": "GraphMem Production Dashboard",
    "panels": [
      {
        "title": "Ingestion Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(graphmem_docs_ingested_total[5m])",
            "legendFormat": "{{tenant_id}}"
          }
        ]
      },
      {
        "title": "Query Latency (p99)",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.99, rate(graphmem_query_latency_seconds_bucket[5m]))",
            "legendFormat": "p99"
          }
        ]
      },
      {
        "title": "Cache Hit Rate",
        "type": "gauge",
        "targets": [
          {
            "expr": "sum(rate(graphmem_cache_hits_total[5m])) / (sum(rate(graphmem_cache_hits_total[5m])) + sum(rate(graphmem_cache_misses_total[5m])))"
          }
        ]
      },
      {
        "title": "Worker Queue Lag",
        "type": "graph",
        "targets": [
          {
            "expr": "graphmem_worker_queue_lag",
            "legendFormat": "{{worker_type}} - {{topic}}"
          }
        ]
      },
      {
        "title": "Active Workers",
        "type": "stat",
        "targets": [
          {
            "expr": "sum(graphmem_workers_active) by (worker_type)"
          }
        ]
      }
    ]
  }
}
```

---

## زیرساخت بنچمارک (سنجش عملکرد)

### اسکریپت بنچمارک در مقیاس بزرگ

```python
# benchmark/distributed_benchmark.py
"""
Large-scale distributed benchmarking for GraphMem.

Tests:
- Ingestion throughput (1M+ documents)
- Query latency under load
- Concurrent user simulation
- Evolution performance
"""

import asyncio
import time
import json
import random
import statistics
from typing import List, Dict, Any
from dataclasses import dataclass, field
from concurrent.futures import ThreadPoolExecutor
import aiohttp

@dataclass
class BenchmarkConfig:
    gateway_url: str = "http://localhost:8000"
    num_documents: int = 100000
    batch_size: int = 1000
    num_concurrent_queries: int = 100
    num_query_iterations: int = 1000
    num_tenants: int = 10

@dataclass
class BenchmarkResults:
    # Ingestion
    ingestion_total_docs: int = 0
    ingestion_elapsed_seconds: float = 0
    ingestion_throughput: float = 0
    ingestion_errors: int = 0
    
    # Query
    query_count: int = 0
    query_latencies: List[float] = field(default_factory=list)
    query_p50_ms: float = 0
    query_p95_ms: float = 0
    query_p99_ms: float = 0
    query_errors: int = 0
    
    # Memory
    total_entities: int = 0
    total_relationships: int = 0
    
    def to_dict(self):
        return {
            "ingestion": {
                "total_docs": self.ingestion_total_docs,
                "elapsed_seconds": self.ingestion_elapsed_seconds,
                "throughput_docs_per_sec": self.ingestion_throughput,
                "errors": self.ingestion_errors,
            },
            "query": {
                "count": self.query_count,
                "p50_ms": self.query_p50_ms,
                "p95_ms": self.query_p95_ms,
                "p99_ms": self.query_p99_ms,
                "errors": self.query_errors,
            },
            "memory": {
                "total_entities": self.total_entities,
                "total_relationships": self.total_relationships,
            }
        }


class DistributedBenchmark:
    """Run large-scale benchmarks against distributed GraphMem."""
    
    def __init__(self, config: BenchmarkConfig):
        self.config = config
        self.results = BenchmarkResults()
    
    def _generate_document(self, idx: int) -> Dict[str, Any]:
        """Generate a realistic test document."""
        companies = ["Acme Corp", "Globex Inc", "Initech", "Hooli", "Piedpiper"]
        people = ["John Smith", "Jane Doe", "Bob Johnson", "Alice Brown", "Charlie Wilson"]
        
        company = random.choice(companies)
        person = random.choice(people)
        revenue = random.randint(1, 100) * 10
        employees = random.randint(100, 10000)
        year = random.randint(2010, 2024)
        
        return {
            "id": f"doc_{idx}",
            "content": f"""
                {company} reported ${revenue}M in revenue for Q{random.randint(1,4)} {year}.
                CEO {person} announced plans to expand the workforce to {employees} employees.
                The company's main focus is on {random.choice(['AI', 'cloud', 'security', 'analytics'])}.
            """,
            "metadata": {
                "source": "benchmark",
                "index": idx,
            }
        }
    
    async def _ingest_batch(
        self,
        session: aiohttp.ClientSession,
        documents: List[Dict],
        tenant_id: str,
    ):
        """Ingest a batch of documents."""
        try:
            async with session.post(
                f"{self.config.gateway_url}/ingest/batch",
                json={
                    "documents": documents,
                    "tenant_id": tenant_id,
                    "memory_id": f"benchmark_{tenant_id}",
                },
                timeout=aiohttp.ClientTimeout(total=300),
            ) as response:
                if response.status == 200:
                    return True
                else:
                    self.results.ingestion_errors += 1
                    return False
        except Exception as e:
            self.results.ingestion_errors += 1
            return False
    
    async def benchmark_ingestion(self):
        """Benchmark document ingestion throughput."""
        print(f"🚀 Starting ingestion benchmark: {self.config.num_documents} documents")
        
        # Generate all documents
        documents = [
            self._generate_document(i) 
            for i in range(self.config.num_documents)
        ]
        
        # Distribute across tenants
        docs_per_tenant = self.config.num_documents // self.config.num_tenants
        
        start_time = time.time()
        
        async with aiohttp.ClientSession() as session:
            tasks = []
            
            for tenant_idx in range(self.config.num_tenants):
                tenant_id = f"tenant_{tenant_idx}"
                tenant_docs = documents[
                    tenant_idx * docs_per_tenant : (tenant_idx + 1) * docs_per_tenant
                ]
                
                # Split into batches
                for i in range(0, len(tenant_docs), self.config.batch_size):
                    batch = tenant_docs[i : i + self.config.batch_size]
                    tasks.append(self._ingest_batch(session, batch, tenant_id))
            
            # Run all ingestion tasks
            results = await asyncio.gather(*tasks)
        
        elapsed = time.time() - start_time
        success_count = sum(1 for r in results if r)
        
        self.results.ingestion_total_docs = self.config.num_documents
        self.results.ingestion_elapsed_seconds = elapsed
        self.results.ingestion_throughput = self.config.num_documents / elapsed
        
        print(f"✅ Ingestion complete: {self.results.ingestion_throughput:.1f} docs/sec")
    
    async def _query(
        self,
        session: aiohttp.ClientSession,
        query: str,
        tenant_id: str,
    ) -> float:
        """Execute a single query and return latency."""
        start = time.time()
        try:
            async with session.post(
                f"{self.config.gateway_url}/query",
                json={
                    "question": query,
                    "tenant_id": tenant_id,
                    "memory_id": f"benchmark_{tenant_id}",
                },
                timeout=aiohttp.ClientTimeout(total=30),
            ) as response:
                if response.status == 200:
                    latency = (time.time() - start) * 1000  # ms
                    self.results.query_latencies.append(latency)
                    return latency
                else:
                    self.results.query_errors += 1
                    return -1
        except Exception:
            self.results.query_errors += 1
            return -1
    
    async def benchmark_queries(self):
        """Benchmark query latency under load."""
        print(f"🔍 Starting query benchmark: {self.config.num_query_iterations} queries")
        
        queries = [
            "What is the company's revenue?",
            "Who is the CEO?",
            "How many employees does the company have?",
            "What is the main focus area?",
            "When was the revenue reported?",
        ]
        
        async with aiohttp.ClientSession() as session:
            # Warm up
            for _ in range(10):
                await self._query(session, random.choice(queries), "tenant_0")
            
            self.results.query_latencies.clear()
            
            # Concurrent queries
            for iteration in range(self.config.num_query_iterations // self.config.num_concurrent_queries):
                tasks = []
                for _ in range(self.config.num_concurrent_queries):
                    tenant_id = f"tenant_{random.randint(0, self.config.num_tenants - 1)}"
                    tasks.append(self._query(session, random.choice(queries), tenant_id))
                
                await asyncio.gather(*tasks)
                
                if (iteration + 1) % 10 == 0:
                    print(f"   Progress: {(iteration + 1) * self.config.num_concurrent_queries} queries")
        
        # Calculate percentiles
        latencies = sorted([l for l in self.results.query_latencies if l > 0])
        if latencies:
            self.results.query_count = len(latencies)
            self.results.query_p50_ms = latencies[int(len(latencies) * 0.50)]
            self.results.query_p95_ms = latencies[int(len(latencies) * 0.95)]
            self.results.query_p99_ms = latencies[int(len(latencies) * 0.99)]
        
        print(f"✅ Query benchmark complete:")
        print(f"   p50: {self.results.query_p50_ms:.1f}ms")
        print(f"   p95: {self.results.query_p95_ms:.1f}ms")
        print(f"   p99: {self.results.query_p99_ms:.1f}ms")
    
    async def run(self):
        """Run complete benchmark suite."""
        print("=" * 60)
        print("GRAPHMEM DISTRIBUTED BENCHMARK")
        print("=" * 60)
        
        await self.benchmark_ingestion()
        await self.benchmark_queries()
        
        print("\n" + "=" * 60)
        print("RESULTS")
        print("=" * 60)
        print(json.dumps(self.results.to_dict(), indent=2))
        
        # Save results
        with open("benchmark_results.json", "w") as f:
            json.dump(self.results.to_dict(), f, indent=2)
        
        return self.results


async def main():
    config = BenchmarkConfig(
        gateway_url="http://localhost:8000",
        num_documents=100000,
        batch_size=1000,
        num_concurrent_queries=100,
        num_query_iterations=10000,
        num_tenants=10,
    )
    
    benchmark = DistributedBenchmark(config)
    await benchmark.run()


if __name__ == "__main__":
    asyncio.run(main())
```

---

## بهینه‌سازی هزینه

### جدول مقایسه هزینه

| مؤلفه | سرویس ابری | هزینه ماهانه | هزینه میزبانی خودی |
| --- | --- | --- | --- |
| **Neo4j** | Aura Enterprise | بیش از ۲٬۵۰۰ دلار | ۵۰۰ دلار (۳× r6g.2xlarge) |
| **Redis** | ElastiCache | بیش از ۵۰۰ دلار | ۲۰۰ دلار (۳× r6g.large) |
| **Kafka** | Confluent Cloud | بیش از ۱٬۰۰۰ دلار | ۳۰۰ دلار (Redpanda روی ۳× m6g.large) |
| **GPU (بردارها)** | فراخوانی API | بیش از ۱۰٬۰۰۰ دلار (۱۰ میلیون سند) | ۱٬۰۰۰ دلار (A100 اسپات) |
| **استخراج با LLM** | OpenAI GPT-4o-mini | ۵٬۰۰۰ دلار (۱۰ میلیون سند) | ۵٬۰۰۰ دلار (مشابه) |
| **محاسبات (K8s)** | EKS | بیش از ۵۰۰ دلار | ۳۰۰ دلار (خودمدیریت) |

### راهکارهای بهینه‌سازی

۱. **استفاده از بردارهای محلی**: ماهانه بیش از ۹٬۰۰۰ دلار در هزینه فراخوانی‌های API صرفه‌جویی می‌کند.
۲. **استفاده از GPT-4o-mini**: برای استخراج ۱۰ برابر ارزان‌تر از GPT-4 است.
۳. **کش Redis**: بیش از ۵۰٪ از فراخوانی‌های تکراری LLM را کاهش می‌دهد.
۴. **پردازش دسته‌ای**: سربار API را تا ۸۰٪ کاهش می‌دهد.
۵. **اینسنس‌های اسپات**: ۷۰٪ در هزینه‌های GPU صرفه‌جویی می‌کند.

---

## چک‌لیست محیط تولید

### قبل از راه‌اندازی

- [ ] **زیرساخت**
  - [ ] خوشه Kubernetes استقرار یافته
  - [ ] خوشه Redpanda/Kafka در حال اجرا
  - [ ] خوشه Neo4j با نسخه‌های تکراری (Replica)
  - [ ] خوشه Redis پیکربندی شده
  - [ ] ذخیره‌سازی اشیاء (S3/MinIO) آماده

- [ ] **امنیت**
  - [ ] TLS/SSL در همه‌جا
  - [ ] چرخش کلید API پیکربندی شده
  - [ ] سیاست‌های شبکه اعمال شده
  - [ ] اسرار در Vault/AWS Secrets Manager

- [ ] **پایش**
  - [ ] Prometheus همه سرویس‌ها را نمونه‌برداری می‌کند
  - [ ] داشبوردهای Grafana استقرار یافته
  - [ ] قوانین هشدار پیکربندی شده
  - [ ] ردیابی توزیع‌شده فعال شده

- [ ] **تست**
  - [ ] تست بار تکمیل شده
  - [ ] تست‌های مهندسی آشوب (Chaos) موفق بوده
  - [ ] تست انتقال به نمونه جایگزین (Failover) انجام شده
  - [ ] پشتیبان‌گیری/بازیابی تأیید شده

### پس از راه‌اندازی

- [ ] **عملیات**
  - [ ] کتابچه‌های عملیاتی (Runbook) مستند شده
  - [ ] نوبت‌بندی پاسخگویی (On-call) تنظیم شده
  - [ ] برنامه واکنش به حوادث آماده است
  - [ ] SLA/SLO تعریف شده

- [ ] **بهینه‌سازی**
  - [ ] مقیاس‌پذیری خودکار تنظیم شده
  - [ ] درخواست‌های منابع بهینه‌سازی شده
  - [ ] نرخ برخورد کش (Cache Hit Rate) پایش می‌شود
  - [ ] داشبورد هزینه فعال است

---

## گام‌های بعدی

۱. **کم‌شروع کنید**: برای توسعه از داکر کامپوز استفاده کنید.
۲. **به‌تدریج مقیاس دهید**: طبق نیاز کارگر اضافه کنید.
۳. **همه‌چیز را پایش کنید**: مشکلات را زودهنگام شناسایی کنید.
۴. **پیوسته بهینه‌سازی کنید**: معیارها را ردیابی و بهبود دهید.

برای پرسش یا پشتیبانی به این منابع مراجعه کنید:
- گزارش مشکلات در گیت‌هاب: https://github.com/Al-aminI/GraphMem/issues
- مستندات: https://github.com/Al-aminI/GraphMem/docs

---

*ساخته‌شده برای سیستم‌های حافظه هوش مصنوعی درجه‌تولید با هر مقیاسی.*
