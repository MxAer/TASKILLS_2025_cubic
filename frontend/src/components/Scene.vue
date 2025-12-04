<template>
  <div class="scene-container">
    <canvas ref="canvasRef"></canvas>
  </div>
</template>

<script setup>
import { onMounted, ref, watch } from 'vue';
import { DiceWorld } from '../core/DiceWorld.js';
import { useGameStore } from '../stores/game';

const canvasRef = ref(null);
const store = useGameStore();
let world = null;

// Слушаем команду на бросок извне (через EventBus или просто через ref, 
// но пока сделаем expose метода для родителя)
defineExpose({
  triggerThrow: (count) => {
    if (world) world.throwDice(count);
  }
});

onMounted(() => {
  if (canvasRef.value) {
    world = new DiceWorld(canvasRef.value);
    world.animate();
    
    // Инициализация темы
    world.setTheme(store.currentTheme);
  }
});

// Реакция на смену темы в сторе
watch(() => store.currentTheme, (newTheme) => {
  if (world) world.setTheme(newTheme);
});
</script>

<style scoped>
.scene-container {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 1; /* Под UI */
  overflow: hidden;
}
canvas {
  display: block;
}
</style>
