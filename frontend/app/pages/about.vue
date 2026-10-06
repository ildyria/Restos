<script setup lang="ts">
import type { AboutPage } from '~~/shared/types'

useSeoMeta({ title: `About us — ${SITE_NAME}`, description: 'Who we are and how we rate restaurants.' })

const { data: about } = await useFetch<AboutPage>('/api/about')
</script>

<template>
  <article class="max-w-2xl">
    <h1 class="font-display text-4xl leading-none sm:text-5xl">About us</h1>

    <img
      v-if="about?.photo"
      :src="mediaUrl(about.photo, 'full')"
      alt="Us"
      class="mt-8 max-h-[28rem] w-full rounded-md object-cover"
    >

    <!-- Rendered by the server from Markdown, without raw HTML. -->
    <div v-if="about" class="prose-journal mt-8 text-lg leading-relaxed text-toned" v-html="about.html" />

    <NuxtLink to="/" class="mt-10 inline-block text-primary hover:underline">See the ranking →</NuxtLink>
  </article>
</template>
