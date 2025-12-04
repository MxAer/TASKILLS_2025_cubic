<template>
  <!-- 3D Scene Background -->
  <Scene ref="sceneRef" />

  <!-- UI Overlay -->
  <div class="ui-overlay">
    
    <!-- Header / Theme Toggle -->
    <header class="header">
      <div class="logo">DICE ROLLER</div>
      <button class="btn btn-small" @click="store.toggleTheme">
        {{ store.currentTheme === 'clean' ? '🔦 MODE: CLEAN' : '☢️ MODE: INDUSTRIAL' }}
      </button>
    </header>

    <!-- Controls Card -->
    <main class="controls-wrapper">
      <div class="card controls">
        <div class="dice-selector">
          <label>TYPE:</label>
          <select v-model="store.diceType" class="input">
            <option value="d6">D6 (Cube)</option>
            <!-- Другие типы добавим позже -->
          </select>
        </div>

        <div class="dice-selector">
          <label>COUNT: {{ store.diceCount }}</label>
          <input type="range" min="1" max="10" v-model="store.diceCount" />
        </div>

        <button class="btn btn-primary" @click="handleThrow">
          ROLL DICE
        </button>
      </div>
    </main>

  </div>
</template>

<script setup>
import { ref } from 'vue';
import Scene from './components/Scene.vue';
import { useGameStore } from './stores/game';

const store = useGameStore();
const sceneRef = ref(null);

const handleThrow = () => {
  store.isRolling = true;
  // Вызываем метод 3D мира через ref
  sceneRef.value.triggerThrow(store.diceCount);
  
  // Фейковый таймаут получения результата (в Stage 4 тут будет WebSocket)
  setTimeout(() => {
    store.isRolling = false;
  }, 2000);
};
</script>

<style>
/* Импортируем глобальные стили и переменные */
@import './style.css';

.ui-overlay {
  position: relative;
  z-index: 10; /* Поверх канваса */
  height: 100vh;
  display: flex;
  flex-direction: column;
  pointer-events: none; /* Чтобы клики проходили сквозь пустоту к канвасу (если нужно вращать камеру) */
}

.header {
  padding: 20px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  pointer-events: auto;
}

.logo {
  font-weight: bold;
  font-size: 1.5rem;
  font-family: var(--font-main);
  color: var(--text-color);
}

.controls-wrapper {
  flex-grow: 1;
  display: flex;
  align-items: flex-end; /* Панель снизу */
  justify-content: center;
  padding-bottom: 50px;
}

.controls {
  padding: 20px;
  width: 300px;
  display: flex;
  flex-direction: column;
  gap: 15px;
  pointer-events: auto;
}

.dice-selector {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-family: var(--font-main);
}

.input {
  background: transparent;
  border: 1px solid var(--border-color);
  color: var(--text-color);
  padding: 5px;
  border-radius: 4px;
}

.btn-primary {
  width: 100%;
  margin-top: 10px;
  font-weight: bold;
}

.btn-small {
  font-size: 0.8rem;
  padding: 5px 10px;
}
</style>
