# Application Email for X (Twitter) - Software Engineer Position

## Email Template (English)

**Subject:** Hardcore Software Engineer - Realtime Timeline Engine Prototype

**To:** careers@x.com (or the appropriate contact)

---

Hi X Hiring Team,

I built a small but production-minded prototype of a realtime social timeline engine that demonstrates my understanding of X's core systems.

**What I built:**
- Realtime tweet delivery via WebSocket
- Hybrid fanout model (fanout-on-write for normal users, fanout-on-read for celebrities)
- Redis-based timeline cache for sub-50ms reads
- Ranking algorithm with time decay and engagement signals
- Rate limiting and spam control
- AI timeline summarization
- Load tested: 10,000+ users, 1M+ tweets, p95 latency <50ms

**Why this matters:**
This touches the real challenges X faces: information velocity, scale, ranking, and trust & safety. I didn't build another Twitter clone UI - I focused on the hard backend problems that make X work at scale.

**GitHub:** [Your GitHub URL]
**Demo:** [Deployed URL if available]

I care more about building useful systems than credentials. Happy to dive deep on any part of the code.

Best,
[Your Name]
[Your Location / Timezone]
[LinkedIn / Portfolio]

---

## 中文版（给中文联系人）

**主题：** 申请软件工程师 - 实时信息流系统原型

X 团队好，

我做了一个实时社交信息流系统的原型，展示我对 X 核心技术的理解。

**核心功能：**
- WebSocket 实时推文
- 混合 Fanout 模型（大 V 用读扩散，普通用户用写扩散）
- Redis 缓存，p95 延迟 <50ms
- 基于时间衰减和互动信号的排序算法
- 限流和防刷机制
- AI 信息流总结

**为什么重要：**
这解决了 X 真正面临的挑战：信息速度、规模、排序、内容安全。我没有做一个 Twitter 的静态页面，而是专注于让 X 能在大规模下运行的后端难题。

**GitHub:** [你的 GitHub 地址]
**Demo:** [部署地址]

比起学历，我更在乎能做出有用的系统。代码的任何部分都可以深入讨论。

[你的名字]

---

## 投递建议

1. **Fork 到 GitHub** 并设置为 Public
2. **部署到免费平台**（Railway, Render, Fly.io）
3. **用 Loom 录一个 2 分钟 demo 视频** 展示功能
4. **LinkedIn 上找招聘者** 直接发消息附链接
5. **邮件标题要硬**：不要 "Application for SWE"，用 "Built a realtime timeline engine - applying for SWE"

## 快速开始命令

```bash
# Backend
cd backend
go mod tidy
docker-compose up -d
go run cmd/server/main.go

# Frontend  
cd frontend
npm install
npm run dev
```
