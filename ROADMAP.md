# Project Context & Roadmap: Ultimate Dice Roller

## 1. Описание проекта и ТЗ

**Цель:** Разработать веб-приложение для броска кубиков с мультиплеером, поддерживающее переключение между кардинально разными визуальными стилями (Clean UI vs Industrial Lo-Fi).

**Проблема:** Отсутствие кубиков под рукой. Скучные интерфейсы генераторов.
**Решение:** Атмосферное веб-приложение + Telegram бот + OBS виджет.

### Функциональные требования (MVP + Maximum)
*   [x] Выбор дайсов (d4-d100).
*   [x] 3D-физика бросков.
*   [ ] Мультиплеер (комнаты, синхронизация).
*   [ ] История и статистика.
*   [x] **Динамическая смена тем оформления.**

---

## 2. Технический стек

*   **Frontend:** Vue 3 (Composition API), Pinia, Three.js + Cannon.js, Vite.
*   **Backend:** Go (Chi), Gorilla WebSocket.
*   **DB/Infra:** PostgreSQL, Redis, Docker.
*   **Bot:** Python (Aiogram 3).

---

## 3. Дизайн-система и Темы

Приложение должно поддерживать горячую смену тем через атрибут `data-theme` на теге `<body>`.

### CSS Variables & Base Styles
В файл стилей нужно включить **оба** набора переменных.

```css
/* ================= THEME ENGINE ================= */
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;700&family=VT323&display=swap');

:root {
    /* DEFAULT THEME (AGRO/CLEAN) */
    --font-main: 'Inter', sans-serif;
    --bg-color: #ffffff;
    --text-color: #1f2937;
    --accent-color: #34d399; /* Emerald */
    --card-bg: #f9fafb;
    --border-width: 1px;
    --border-color: #e5e7eb;
    --shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.1);
    --crt-overlay: none;
    --radius: 1rem;
    --btn-transform: uppercase;
}

/* DARK MODE (CLEAN) */
@media (prefers-color-scheme: dark) {
    :root {
        --bg-color: #111827;
        --text-color: #f3f4f6;
        --card-bg: #1f2937;
        --border-color: #374151;
    }
}

/* ================= THEME: KLUBNIKA / INDUSTRIAL ================= */
/* Стиль Buckshot Roulette: Грязь, Металл, CRT монитор */
[data-theme="industrial"] {
    --font-main: 'VT323', monospace; /* Pixelated font */
    --bg-color: #0c0a08; /* Rusty Dark */
    --text-color: #d6d3d1; /* Old paper */
    --accent-color: #ea580c; /* Rusty Orange / Industrial Amber */
    --card-bg: #1c1917;
    --border-width: 2px;
    --border-color: #44403c;
    --shadow: 4px 4px 0px #000000; /* Hard shadow, no blur */
    --radius: 0px; /* Brutalist corners */
    --btn-transform: uppercase;
    
    /* CRT Effect Overlay */
    --crt-overlay: linear-gradient(rgba(18, 16, 16, 0) 50%, rgba(0, 0, 0, 0.25) 50%), linear-gradient(90deg, rgba(255, 0, 0, 0.06), rgba(0, 255, 0, 0.02), rgba(0, 0, 255, 0.06));
}

/* GLOBAL STYLES APPLIED TO VUE */
body {
    background-color: var(--bg-color);
    color: var(--text-color);
    font-family: var(--font-main);
    transition: all 0.3s ease;
    overflow: hidden;
}

/* CRT Scanlines for Industrial Theme */
body::after {
    content: " ";
    display: block;
    position: absolute;
    top: 0; left: 0; bottom: 0; right: 0;
    background: var(--crt-overlay);
    background-size: 100% 2px, 3px 100%;
    z-index: 9999;
    pointer-events: none;
}

.btn {
    background: var(--card-bg);
    border: var(--border-width) solid var(--accent-color);
    color: var(--accent-color);
    font-family: var(--font-main);
    text-transform: var(--btn-transform);
    cursor: pointer;
    border-radius: var(--radius);
    padding: 10px 20px;
    font-size: 1.2rem;
}

[data-theme="industrial"] .btn:hover {
    background: var(--accent-color);
    color: #000;
    box-shadow: 0 0 10px var(--accent-color); /* Glow effect */
}

.card {
    background: var(--card-bg);
    border: var(--border-width) solid var(--border-color);
    border-radius: var(--radius);
    box-shadow: var(--shadow);
}
```

---

## 4. Roadmap (11 Stages)

### Stage 1: Initialization & Infrastructure
*Goal: Поднять окружение (Docker, Repos).*
- [x] Создать структуру монорепозитория (`/frontend`, `/backend`, `/bot`).
- [x] Настроить `docker-compose.yml` (Postgres, Redis, Go-backend, Vue-frontend).
- [x] Настроить базовый Go module (Chi router) и Vue 3 project (Vite).
- [x] Настроить Linter и `.editorconfig` для единого стиля кода.

### Stage 2: Backend Core & Database
*Goal: API для создания комнат.*
- [x] Спроектировать SQL схему: `rooms` (UUID), `users`, `rolls` (JSONB).
- [x] Написать Go-миграции для PostgreSQL.
- [x] Реализовать REST API: `POST /api/room`, `GET /api/room/{id}`.
- [x] Реализовать регистрацию "гостя" (Session ID в cookie).

### Stage 3: Frontend Core & 3D Physics
*Goal: Базовая механика броска.*
- [x] Инициализировать Three.js сцену + Cannon.js мир.
- [x] Создать 3D модели дайсов (d4, d6, d8, d10, d12, d20).
- [x] Реализовать логику: Нажал кнопку -> Импульс физике -> Ожидание остановки -> Чтение грани.
- [x] Сверстать базовый UI (выбор дайсов) используя CSS Variables.

### Stage 4: Real-time Multiplayer (WebSocket)
*Goal: Синхронизация игроков.*
- [ ] **Backend:** Реализовать WebSocket Hub (Upgrader, Client management).
- [ ] **Redis:** Настроить Pub/Sub для передачи сообщений между инстансами бэкенда.
- [ ] **Frontend:** Написать WebSocket Client (Pinia Store).
- [ ] Синхронизировать событие `THROW_START` (вектор силы) и `THROW_RESULT` (финальные координаты).

### Stage 5: Telegram Bot Integration
*Goal: Вход через Телеграм.*
- [ ] Написать бота на Aiogram 3 (`/start`, `/new_game`).
- [ ] Реализовать WebApp кнопку (открывает фронтенд внутри TG).
- [ ] Настроить передачу `tg_user_id` и `username` в URL WebApp.
- [ ] Бот должен слушать Redis канал комнаты и писать в чат: "🎲 @User выбросил 18!".

### Stage 6: Theming Engine & "Klubnika" Mode 🆕
*Goal: Реализация переключения стилей.*
- [ ] Создать Vue Composable `useTheme()` для управления `document.body.dataset.theme`.
- [ ] Реализовать UI переключатель: "Clean Mode" / "Industrial Mode".
- [ ] **Three.js Adaptation:**
    *   Если тема Clean: Использовать светлое окружение, мягкие тени, цветные кубики.
    *   Если тема Industrial: Выключать глобальный свет, ставить SpotLight (эффект фонарика), менять текстуры кубиков на "ржавый металл", добавить пост-процессинг (Grain/Noise) на канвас.
- [ ] Добавить звуки для темы Industrial (тяжелые удары, гул электричества).

### Stage 7: Advanced Game Mechanics
*Goal: Углубление геймплея.*
- [ ] Реализовать "Скрытый бросок" (GM Roll).
- [ ] Реализовать кастомные формулы (парсинг строки `2d20+5`).
- [ ] Сохранять историю бросков в БД и выводить список на фронте.

### Stage 8: Immersive Features
*Goal: "Вау-эффекты".*
- [ ] **Mobile:** Shake API (бросок тряской телефона).
- [ ] **Haptics:** Вибрация при ударах кубиков.
- [ ] **AR Mode (WebXR):** Базовая реализация отображения кубиков на камере (по возможности).

### Stage 9: Analytics & AI
*Goal: Аналитика.*
- [ ] Реализовать "Heatmap" стола (где чаще останавливаются кубики).
- [ ] Подсчет "Удачи" (сравнение среднего броска с матжиданием).
- [ ] AI-Stub: Функция-заглушка для генерации текста описания броска.

### Stage 10: Streaming & OBS Widget
*Goal: Инструменты для стримеров.*
- [ ] Создать отдельный лейаут `/overlay/{roomId}` (прозрачный фон).
- [ ] Убрать весь UI управления, оставить только 3D сцену и всплывающий результат.
- [ ] Оптимизировать шрифты для читаемости на стриме.

### Stage 11: CI/CD & Final Polish
*Goal: Релиз.*
- [ ] Написать E2E тесты (Playwright) для переключения тем и бросков.
- [ ] Настроить GitHub Actions (Lint -> Test -> Build -> Docker Push).
- [ ] Оптимизировать бандл (Lazy loading для текстур и моделей).
- [ ] Создать `k8s` манифесты для деплоя.

---

**Инструкция для AI:**
Генерируй код строго по этим этапам. При реализации Frontend **всегда** используй CSS переменные для цветов, чтобы переключение тем (Clean <-> Industrial) работало автоматически. Для Three.js сцены проверяй текущую тему и меняй освещение/материалы соответственно.