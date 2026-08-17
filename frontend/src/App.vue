<script setup lang="ts">
import ToastHost from './components/ToastHost.vue'
import SiteFooter from './components/SiteFooter.vue'
</script>
<template>
  <div class="app-frame">
    <div class="app-main">
      <router-view v-slot="{ Component }">
        <transition name="page" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </div>
    <SiteFooter />
  </div>
  <ToastHost />
</template>
