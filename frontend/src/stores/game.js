import { defineStore } from 'pinia';
import { ref } from 'vue';

export const useGameStore = defineStore('game', () => {
    const diceType = ref('d6');
    const diceCount = ref(2);
    const lastResult = ref(null);
    const isRolling = ref(false);

    // Тема приложения (связана с CSS переменными через DOM)
    const currentTheme = ref('clean');

    function toggleTheme() {
        currentTheme.value = currentTheme.value === 'clean' ? 'industrial' : 'clean';
        // Обновляем атрибут на body для CSS
        if (currentTheme.value === 'industrial') {
            document.body.setAttribute('data-theme', 'industrial');
        } else {
            document.body.removeAttribute('data-theme');
        }
    }

    return { diceType, diceCount, lastResult, isRolling, currentTheme, toggleTheme };
});
