<template>
  <div class="rain-container" aria-hidden="true" v-if="mounted">
    <div
      v-for="d in drops"
      :key="d.id"
      class="drop"
      :style="d.style"
    ></div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'

const mounted = ref(false)
const drops = ref([])

onMounted(() => {
  // Generate drops with depth simulation
  drops.value = Array.from({ length: 60 }, (_, i) => {
    const left = Math.floor(Math.random() * 100);
    const delay = (Math.random() * 5).toFixed(2);
    
    // Depth factor (0 = far, 1 = near)
    // Near drops: faster, more opaque, longer
    // Far drops: slower, more transparent, shorter
    const depth = Math.random(); 
    
    const duration = (1.5 + (1 - depth) * 1.5).toFixed(2); // 1.5s (near) to 3.0s (far)
    const opacity = (0.1 + depth * 0.3).toFixed(2); // 0.1 (far) to 0.4 (near)
    
    return {
      id: i,
      style: {
        left: `${left}%`,
        animationDelay: `-${delay}s`,
        animationDuration: `${duration}s`,
        opacity: opacity,
        // Scale affecting width mainly, height is handled by CSS
        transform: `scale(${0.8 + depth * 0.4})` 
      }
    }
  })
  mounted.value = true
})
</script>

<style scoped>
.rain-container {
  position: fixed;
  top: -20vh; /* Start higher to cover rotation gaps */
  left: -20vw; /* Start wider */
  width: 140vw; /* Significantly wider than viewport to handle slant */
  height: 140vh;
  pointer-events: none;
  z-index: -1; /* Strictly behind all content */
  overflow: hidden;
  /* We rotate the container itself for the wind direction, 
     simplifying the drop animation to just 'fall down' relative to container */
  transform: rotate(15deg); 
}

.drop {
  position: absolute;
  top: 0;
  width: 1px;
  height: 30vh; /* Long streaks */
  background: linear-gradient(180deg, 
    rgba(255, 255, 255, 0) 0%, 
    rgba(255, 255, 255, 0.4) 50%, 
    rgba(255, 255, 255, 0) 100%
  );
  will-change: transform, opacity;
  /* Animation handles the movement */
  animation: fall linear infinite;
}

/* Dark mode: softer blue-grey for rain */
:global(.dark) .drop {
  background: linear-gradient(180deg, 
    rgba(100, 116, 139, 0) 0%, 
    rgba(56, 189, 248, 0.3) 50%, 
    rgba(100, 116, 139, 0) 100%
  );
}

@keyframes fall {
  0% {
    transform: translateY(-30vh);
  }
  100% {
    transform: translateY(130vh);
  }
}
</style>
