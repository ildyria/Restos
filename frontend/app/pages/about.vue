<script setup lang="ts">
import type { AboutPage } from '~~/shared/types'

useSeoMeta({ title: `About us — ${SITE_NAME}`, description: 'Who we are and how we rate restaurants.' })

const { data: about } = await useFetch<AboutPage>('/api/about')
</script>

<template>
  <!-- A centred reading column; on large screens the photo sits beside the text instead of above it. -->
  <article class="mx-auto max-w-2xl" :class="about?.photo && 'lg:max-w-none'">
    <h1 class="font-display text-4xl leading-none sm:text-5xl">About us</h1>

    <div class="mt-8" :class="about?.photo && 'lg:grid lg:grid-cols-2 lg:items-start lg:gap-12'">
      <img
        v-if="about?.photo"
        :src="mediaUrl(about.photo, 'full')"
        alt="Us"
        class="mb-8 max-h-[28rem] w-full rounded-md object-cover lg:sticky lg:top-8 lg:order-last lg:mb-0 lg:max-h-[80vh]"
      >

      <div>
        <!-- Rendered by the server from Markdown, without raw HTML. -->
        <div v-if="about" class="prose-journal text-lg leading-relaxed text-toned" v-html="about.html" />

        <NuxtLink to="/" class="mt-10 inline-block text-primary hover:underline">See the ranking →</NuxtLink>
      </div>
    </div>
  </article>
</template>
