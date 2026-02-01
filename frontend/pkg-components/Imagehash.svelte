<script lang="ts">
    import { onMount } from "svelte";
import s1 from './styles.module.css';

import { Env } from '$core/env';

	export interface IImageHash {
			src: string, css: string, hash: string, alt: string, size: number, folder?: string
	}

  const {
  	src, css, hash, alt, size = 4, folder
  }: IImageHash = $props();

  let imageSrc: string | undefined = $state();
  let placeholderSrc = $state("");

  if (hash?.length > 0) {
    placeholderSrc = "/?"+hash;
  }

  onMount(() => {
    if (hash?.length > 0) {
    	let imageName = hash.substring(0, 12).replaceAll(".", "/").replaceAll("-", "=") +
     ".webp"
     	
      imageSrc = Env.makeCDNRoute("images", imageName)
    } else if (src) {
      const sl = src.split(".")
      const ext = sl[sl.length - 1]
      let imageName = src
      if(sl.length < 2 || !["jpeg","jpg","webp","avif","png"].includes(ext)){
        imageName += `-x${size}.avif`
      }
      imageSrc = Env.makeCDNRoute(folder as string, imageName)
    }
    console.log("image source::", imageSrc,"| folder", folder,"| src",src)
	})
</script>

<div class={[s1.image_hash_ctn, css || ""].join(" ")}>
  {#if !!placeholderSrc}
    <img role={`0/0/${src}`} class="image_hash" loading="lazy" alt="" />
  {/if}
  <img role={`1/${size}/${src}`} class="image_hash" src={imageSrc} {alt} loading="lazy"
    onload={() => { placeholderSrc = "" }}
  />
</div>

<style>
  .image_hash {
    width: 100%;
    height: 100%;
    object-fit: contain;
  }
</style>
