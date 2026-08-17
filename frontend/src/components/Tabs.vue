<script setup lang="ts">
type Tab = { id: string; label: string }
const props = defineProps<{ items: Tab[] }>()
const model = defineModel<string>()
</script>
<template><nav class="tabs" aria-label="Settings tabs"><button v-for="item in props.items" :key="item.id" :class="{active:model===item.id}" @click="model=item.id">{{ item.label }}</button></nav></template>
