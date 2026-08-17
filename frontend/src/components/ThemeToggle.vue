<script setup lang="ts">
import { useThemeStore } from '../stores/theme'
const theme = useThemeStore()
theme.apply()
</script>
<template><select aria-label="Theme" :value="theme.theme" @change="theme.setTheme(($event.target as HTMLSelectElement).value as 'light'|'dark'|'system')"><option value="system">System</option><option value="light">Light</option><option value="dark">Dark</option></select></template>
