<script setup lang="ts">
import ToastHost from './components/ToastHost.vue'
</script>
<template>
  <router-view v-slot="{ Component }">
    <transition name="page" mode="out-in">
      <component :is="Component" />
    </transition>
  </router-view>
  <ToastHost />
</template>