<script lang="ts">
    import { onMount } from "svelte";

import { Env } from '$core/env';
    import { derived } from "svelte/store";

	export interface IImageHash {
			src: string, css: string, hash?: string, alt?: string, size?: number, folder?: string
	}

  const {
  	src, css, hash, alt, size = 4, folder
  }: IImageHash = $props();


  let imageSrc = $derived.by(() => {
	  if (hash && hash.length > 0) {
	  	let imageName = hash.substring(0, 12).replaceAll(".", "/").replaceAll("-", "=") +
	   ".webp"
	   	
	    return Env.makeCDNRoute(folder as string, imageName)
	  } else if (src) {
	    const sl = src.split(".")
	    const ext = sl[sl.length - 1]
	    let imageName = src
	    if(sl.length < 2 || !["jpeg","jpg","webp","avif","png"].includes(ext)){
	      imageName += `-x${size}.avif`
	    }
	    return Env.makeCDNRoute(folder as string, imageName)
	  }
  })
  
  let placeholderSrc = $derived.by(() => {
    return (hash && hash.length > 0) ? "/?"+hash : ""
  })
  onMount(() => {
    if (hash && hash.length > 0) {
        let imageName = hash.substring(0, 12).replaceAll(".", "/").replaceAll("-", "=") +
     ".webp"
     	
      imageSrc = Env.makeCDNRoute(folder as string, imageName)
    } else if (src) {
      const sl = src.split(".")
      const ext = sl[sl.length - 1]
      let imageName = src
      if(sl.length < 2 || !["jpeg","jpg","webp","avif","png"].includes(ext)){
        imageName += `-x${size}.avif`
      }
      imageSrc = Env.makeCDNRoute(folder as string, imageName)
    }
    console.log(hash, "image source::", imageSrc,"| folder", folder,"| src",src)
	})
</script>

<div class={["image_hash_ctn", css || ""].join(" ")}>
	{#if !!placeholderSrc}
		<img role={`0/0/${src}`} class="image_hash" loading="lazy" alt="" />
	{/if}
	<img
		role={`1/${size}/${src}`}
		class="image_hash"
		src={imageSrc}
		{alt}
		loading="lazy"
		onload={() => {
			placeholderSrc = "";
		}}
	/>
</div>

<style>
	.image_hash_ctn {
		position: relative;
		background-color: white;
	}
	.image_hash_ctn img {
		position: absolute;
		top: 0;
		left: 0;
		height: 100%;
		width: 100%;
		object-fit: contain;
		border: none;
		outline: none;
	}
	.image_hash_ctn img:last-of-type {
		z-index: 2;
	}
	.image_hash {
		width: 100%;
		height: 100%;
		object-fit: contain;
	}
</style>
