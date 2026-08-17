<script setup lang="ts">
type Props = { variant?: 'primary' | 'secondary' | 'ghost'; type?: 'button' | 'submit' }
const props = withDefaults(defineProps<Props>(), { variant: 'primary', type: 'button' })
</script>
<template><button :type="props.type" class="base-button" :class="`button-${props.variant}`"><slot /></button></template>
