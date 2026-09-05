# AutoGet Organizer Go 重构架构设计文档 (Design Doc)

## 1. 背景与核心设计定位

### 1.1 历史背景与重构痛点
AutoGet Organizer 是自动化媒体处理链路中的后处理核心服务，负责将下载好的各类文件（电影、剧集、书籍、音乐、摄影集、AV/JAV 等）按规范（如 Jellyfin 媒体库结构、女优归档目录等）进行智能重命名与归档。

两年前原系统的根本痛点：
1. **老旧架构为规避 JSON 幻觉而极度臃肿**：两年前 LLM 无法可靠输出复杂结构，原项目堆叠了 9 个单项探测 Agent 串行试探，再加一个 `decision_maker` 猜冲突结果。
2. **在“规划与重命名”阶段使用了脆弱的模式匹配**：
   - 原代码在后续规划阶段试图用大量正则和硬编码去解析季/集（如 `sXXeYY`）、切分分卷、判断字幕与视频关联；
   - **现实痛点**：实际下载的文件名经常带有压制组杂音（`[HDSky]`, `1080p.WEB-DL.DDP5.1.Atmos`）、奇形怪状的集数标号（`第03话`、`1x05`、`OVA1`、`[01-02合集]`）、甚至上游传入的 JSON 字段缺失或拼写错误。一旦靠代码做模式匹配，极易误判或崩溃。
3. **Python 运行时代价**：FastAPI + pydantic-ai 资源占用高，启动慢。

### 1.2 核心架构定位：**前端轻量模式初筛，后续规划与实体解析全部交由 LLM 接管**

> **1. 仅在“阶段一（分类器）”做轻量模式匹配，失败/歧义降级 LLM**：
> - 仅在最前置的分类器做快速规则初判（例如：纯 `.epub/.pdf` 必定是图书；纯 `.flac/.mp3` 必定是音乐；标准番号必定是 JAV）。
> - 若输入不规范、多后缀混杂或存在歧义，立即降级由分类 LLM 裁定。
>
> **2. 阶段一之后（元数据理解、文件规划、重命名、集数解析、字幕对齐）全面交由 LLM 接管**：
> - 后续所有涉及**文件语义理解、杂乱文件名提取清洗、剧集集数映射（SxxExx）、分卷判定（part.1）、字幕与视频配对**的工作，**坚决不写脆弱的硬编码正则与模式匹配**，全部由 LLM 的强大容错与推理能力接管。
> - 代码层仅负责纯粹的流程调度、外部元数据工具调用与底层物理 I/O 安全校验。
>
> **3. 依托现代原生 Structured Outputs（Grok & Gemini）**：
> - 所有 LLM 调用均配置严格的 **JSON Schema / Structured Outputs**，彻底告别过去老系统的格式崩溃与繁琐的防幻觉提示词，保证强类型安全返回。

---

## 2. 总体架构与多阶段处理流水线

整体流水线严格划分为 **4 个职责清晰的正交阶段**：

```
[原始输入 (包含脏文件名 / 压制组杂音 / 上游异常 JSON 字段)]
                         │
                         ▼
┌────────────────────────────────────────────────────────┐
│ 阶段一：分类器 (Stage 1: Classifier)                     │
│  ├─ [模式匹配初筛] 纯后缀(书/音乐) / 强规范番号 -> 命中 │ ───┐
│  └─ [LLM 兜底]    脏输入 / 格式混杂 / 歧义 -> LLM 判定  │    │
└────────────────────────────────────────────────────────┘    │
                         │                                    │
                         ▼ (确定 Category 与初筛实体)           │
┌────────────────────────────────────────────────────────┐    │ (对于 book/music 等
│ 阶段二：定向元数据检索与增强 (Stage 2: Metadata Enricher) │    │  简易类型跳过 LLM 规划)
│  ├─ 依据类别与实体，定向调用外部数据源 (TMDB/JAVDB/MCP) │    │
│  └─ JAV 女优别名库持久化比对 (actor.json + 进程文件锁)  │    │
└────────────────────────────────────────────────────────┘    │
                         │                                    │
                         ▼ (富集后的上下文与全量文件列表)       │
┌────────────────────────────────────────────────────────┐    │
│ 阶段三：专业领域规划器 (Stage 3: LLM Planners)          │    │
│  ★ 全部交由 LLM 语义理解接管，坚决不写集数与分卷正则匹配 ★│    │
│  ├─ TV Series Planner LLM (智能消化任意剧集集数命名)    │    │
│  ├─ Movie Planner LLM (电影正片/花絮识别与 Jellyfin 命名) │    │
│  ├─ Bango/JAV Planner LLM (分卷 part.1 / -C 中文字幕识别)│    │
│  └─ Simple Planner (仅针对图书/音乐相册的直通归档)       │ ◄──┘
└────────────────────────────────────────────────────────┘
                         │
                         ▼ (视频与基础文件规划方案)
┌────────────────────────────────────────────────────────┐
│ 阶段四：伴生字幕配对与物理安全校验 (Stage 4: Post-Process)│
│  ├─ [LLM 语义对齐] 字幕与视频关联配对 (无死板前缀匹配)  │
│  └─ [物理安全校验] 路径防穿越校验 (TARGET_DIR 白名单)     │
└────────────────────────────────────────────────────────┘
                         │
                         ▼
             [最终执行方案 PlanResponse]
```

---

## 3. 各阶段深度设计与职责划分

### 3.1 阶段一：分类器 (Stage 1: Classifier) —— 仅此阶段做模式匹配

- **职责**：只确定媒体分类（`Category`）与初始特征，不处理后续具体文件的移动规划。
- **模式匹配规则（极简、高置信度）**：
  1. **纯后缀文件集合**：
     - 若全量文件扩展名均为电子书格式（`.epub`, `.pdf`, `.mobi`, `.azw3`）-> 直接判定为 `book`；
     - 若全量文件扩展名均为音频格式（`.mp3`, `.flac`, `.wav`, `.ape`）-> 直接判定为 `music`；
  2. **强特征命名**：
     - 单一主视频且符合标准番号格式（`^[A-Z]{3,5}-\d{3,4}$`、`^FC2-PPV-\d+$`）-> 直接判定为 `bango_porn`；
  3. **明确的外部 ID**：上游自带有效 `dmm_id` -> `bango_porn`。
- **LLM 兜底分类（模式匹配未命中时触发）**：
  - **触发场景**：文件名含有大量压制组噪音（如 `[HDSky] The.Matrix...`）、混杂类型（又有视频又有大量图片，无法分清是相册附赠小视频还是影视）、或者上游 JSON 字段有误/拼写错误；
  - **LLM 行为**：使用通用分类 Prompt + **严格 JSON Schema** 输出结构化分类结果，顺便纠偏上游错误（如提取正确的 `imdb_id` 或 `clean_title`）。

### 3.2 阶段二：定向元数据检索与增强 (Stage 2: Metadata Enrichment)

- **职责**：只负责将阶段一确定的类别与实体，补齐为权威的媒体元数据，不介入文件如何移动。
- **调度策略**：
  - **影视类 (`movie` / `tv_series`)**：
    - 若已有 `imdb_id` -> 直接调用 TMDB/IMDb 详情；
    - 若无 `imdb_id` -> 使用阶段一产出的名称搜索；
    - 补全：权威简体中文名称、剧集分季集数详情、官方上映年份。
  - **番号类 (`bango_porn`)**：
    - 调用 `search_japanese_porn` 补全女优列表、制片商、VR 标记；
    - 读取本地 `actor.json`（带 Go 进程文件排他锁 `flock`），匹配女优归档目录；若遇新女优，则进行别名检索并持久化。
  - **简易类别 (`book`, `music` 等)**：跳过本阶段。

### 3.3 阶段三：专业领域规划器 (Stage 3: LLM Domain Planners) —— 全部由 LLM 接管

> **核心决定**：从这里开始，**代码完全放弃任何正则模式匹配**。所有涉及“文件名理解、压制组信息剥离、季集数匹配、分卷抽取”的工作，全部交由 LLM 强大的上下文语义理解能力处理。

#### 3.3.1 剧集规划器 (TV Series Planner LLM)
- **痛点**：真实下载中，剧集集数命名千奇百怪：`第03话`、`1x05`、`EP02`、`[01-02合集]`、`OVA1`、`SP02`、`E01v2`。用代码模式匹配永远修不完 edge-cases。
- **LLM 接管**：
  - 输入：待处理文件列表 + 阶段二获取的剧集官方中文名与年份；
  - Prompt 注入 Jellyfin 命名规范：`tv_series/{Lang}/{中文名 (年份)}/Season {XX}/{中文名 (年份)} S{XX}E{YY}.{ext}`；
  - LLM 依靠语义理解，精准将任意杂乱文件名映射至标准 `S{XX}E{YY}`，完全杜绝正则误判。

#### 3.3.2 电影规划器 (Movie Planner LLM)
- **痛点**：一个文件夹里经常混杂主电影文件、预告片（trailer）、无用样品（sample）、压制说明文件。
- **LLM 接管**：
  - LLM 根据文件名和大小语义识别正片并更名为 `movie/{Lang}/{中文名 (年份)}/{中文名 (年份)}.{ext}`；
  - 自动将 sample、trailer、无用广告视频标记为 `skip`。

#### 3.3.3 番号规划器 (Bango Planner LLM)
- **痛点**：多分卷（如 `cd1/cd2`、`partA/partB`、`上卷/下卷`）、中文字幕后缀（`-C`、`_ch`、`【中文字幕】`）。
- **LLM 接管**：
  - 统一由 LLM 理解分卷并规范化为 `BANGO.part.1.ext`；
  - 识别是否有中文翻译语义，统一保留规范的 `BANGO-C.ext` 命名；
  - 输出格式：`jav/{女优名}/{BANGO}.{ext}`。

#### 3.3.4 简易品类规划 (Simple Planner)
- 仅图书（`book`）、音乐（`music`）、摄影集（`photobook`）等，无需复杂重命名，由本地代码直接归档到一级目录。

### 3.4 阶段四：伴生字幕配对与物理安全校验 (Stage 4: Post-Process) —— LLM 语义配对

- **职责**：字幕智能对齐、无用垃圾文件清理、底层物理路径安全。
- **字幕智能对齐（LLM 语义接管）**：
  - 绝不使用死板的文件名前缀正则字符串切片；
  - 将待处理的字幕文件列表及其前 30 行文本内容，与阶段三规划好的视频列表一同输入 LLM；
  - LLM 语义分析字幕属于哪一集、哪部电影，并识别其语种（简中/繁中/英文/双语），直接输出规范的更名：`<VideoBaseName>.<Language>.<ISO639-2>.<ext>`（例如：`黑客帝国 (1999).简体中文.chi.srt`）。
- **物理安全底线校验（纯 Go 代码）**：
  - 强制对所有输出的 `target` 执行 `filepath.Clean`，必须处于 `TARGET_DIR` 相对目录之内，严防 `../` 路径穿越注入；
  - 将 `.nfo`, `.url`, 种子文件等标记为 `skip`。

---

## 4. LLM 抽象层与 Grok / Gemini 双引擎支持

在所有 LLM 环节，全面采用当前最强力的 **Structured Outputs (严格 JSON Schema)**，确保模型生成的数据能够 100% 毫无偏差地 Unmarshal 进 Go 强类型结构体：

### 4.1 统一 Provider 抽象接口

```go
package ai

import "context"

type Provider interface {
	Name() string
	// GenerateStructured 统一根据 Prompt 与结构体生成的 JSON Schema，输出强类型对象
	GenerateStructured(ctx context.Context, prompt string, schema any, result any) error
}
```

### 4.2 Gemini Provider (Google GenAI)
- **接入**：Google 官方 Go SDK (`google.golang.org/genai`) 或 REST 客户端。
- **配置**：
  - `response_mime_type: "application/json"`
  - `response_schema: schema`（Go 结构体自动转换的 OpenAPI Schema）
  - `temperature: 0.1`

### 4.3 Grok Provider (xAI)
- **接入**：OpenAI 官方兼容 Go SDK 或 HTTP Client 连接 `https://api.x.ai/v1`。
- **配置**：
  - `response_format: { type: "json_schema", json_schema: { name: "output", strict: true, schema: schema } }`
  - `temperature: 0.1`

---

## 5. 项目工程结构组织 (Go)

```
organizer/
├── cmd/
│   └── server/
│       └── main.go                 # 服务入口、环境检查、依赖注入与 HTTP 启动
├── internal/
│   ├── config/                     # 配置加载与启动校验
│   │   └── config.go
│   ├── model/                      # 领域实体与各阶段强类型 DTO
│   │   ├── api.go                  # /v1/plan, /v1/execute 请求与响应结构
│   │   ├── category.go             # 分类枚举定义
│   │   └── stage_dto.go            # 阶段一至阶段四严格数据契约
│   ├── ai/                         # LLM Provider 适配层
│   │   ├── provider.go             # 统一接口定义
│   │   ├── schema.go               # Go 结构体转 JSON Schema 工具
│   │   ├── gemini/                 # Google Gemini 实现
│   │   └── grok/                   # xAI Grok 实现
│   ├── pipeline/                   # 多阶段核心流水线
│   │   ├── pipeline.go             # 流水线协调调度器 (Orchestrator)
│   │   ├── stage1_classifier/      # 【阶段一】仅此处做模式匹配初筛 + LLM 兜底
│   │   │   ├── matcher.go          # 极简高置信度模式匹配 (纯后缀/标准番号)
│   │   │   └── classifier_llm.go   # 脏数据/异常格式 LLM 语义分类器
│   │   ├── stage2_enricher/        # 【阶段二】定向元数据检索与本地演员库维护
│   │   │   ├── enricher.go         # 外部元数据工具调用
│   │   │   └── actor_store.go      # 带文件排他锁的 actor.json 读写
│   │   ├── stage3_planner/         # 【阶段三】专业领域规划器 (全部 LLM 接管)
│   │   │   ├── router.go           # 品类规划路由分发
│   │   │   ├── tv_planner.go       # 剧集专属 LLM (语义解析集数)
│   │   │   ├── movie_planner.go    # 电影专属 LLM (正片/花絮识别)
│   │   │   ├── bango_planner.go    # 番号专属 LLM (分卷/-C识别)
│   │   │   └── simple_planner.go   # 简易品类纯本地直通
│   │   └── stage4_postprocess/     # 【阶段四】伴生资源配对与物理安全校验
│   │       ├── subtitle_llm.go     # LLM 语义字幕配对 (内容/语种识别)
│   │       └── security.go         # 纯代码物理路径防穿越检查
│   ├── service/
│   │   └── executor.go             # 文件物理原子移动与归档 (POST /v1/execute)
│   ├── mcp/                        # MCP 客户端 (与外部元数据服务通信)
│   │   └── client.go
│   └── handler/                    # REST 路由 (完全兼容现有协议)
│       ├── plan_handler.go         # POST /v1/plan
│       ├── execute_handler.go      # POST /v1/execute
│       └── replan_handler.go       # POST /v1/replan-with-hint
├── go.mod
├── go.sum
├── Dockerfile                      # 多阶段 Go 镜像 (~25MB)
└── justfile                        # 自动化任务 (fmt, lint, test, build)
```

---

## 6. 与现有系统 API 兼容性规范

对外保持 100% 协议契约一致，上游 AutoGet 无需任何改动：

1. **`POST /v1/plan`**：
   - 输入：`{"dir": "...", "files": [...], "metadata": {...}}`
   - 内部：流转 **阶段一（模式初筛/LLM兜底） -> 阶段二（元数据富集） -> 阶段三（LLM 语义规划） -> 阶段四（LLM 字幕配对与物理安全校验）**。
   - 输出：`{"plan": [{"file": "...", "action": "move|skip", "target": "..."}], "error": null}`。
2. **`POST /v1/execute`**：
   - 输入：`{"dir": "...", "plan": [...]}`
   - 行为：物理校验 -> `os.MkdirAll` -> `os.Rename` 原子移动 -> 源目录归档至 `DOWNLOAD_COMPLETED_DIR/archive/{dir}`。
3. **`POST /v1/replan-with-hint`**：
   - 输入：`{"files": [...], "metadata": {...}, "previous_response": {...}, "user_hint": "..."}`
   - 行为：将前次规划与用户提示直接注入阶段三的领域 LLM 重新生成规划。

---

## 7. 演进路线与实施计划

1. **第一阶段：模型与契约初始化**
   - 编写 `internal/model`，定义各阶段的数据结构与严格 JSON Schema。
2. **第二阶段：双 Provider 落地**
   - 接入 Gemini 与 Grok，验证其在复杂集数、分卷、脏输入测试用例下的严格输出能力。
3. **第三阶段：各流水线阶段落地**
   - 实现阶段一（模式匹配初筛 + LLM 兜底）；
   - 实现阶段二（元数据检索 + 带锁演员库）；
   - 实现阶段三（剧集、电影、番号专有 LLM 规划器）；
   - 实现阶段四（字幕 LLM 语义对齐 + 路径防穿越）。
4. **第四阶段：物理执行器与全流程集成测试**
   - 编写 `executor.go` 文件原子移动与归档。
   - 使用包含各种压制组杂音、脏元数据的真实样本回归测试。
