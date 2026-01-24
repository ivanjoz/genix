<script lang="ts">
    import { onMount } from "svelte";
  import { Env } from "$lib/security";
  import s1 from "./styles.module.css";

  export let src = "";
  export let css = "";
  export let hash = "";
  export let alt = "";
  export let size = 4;
  let imageSrc: string | undefined = undefined;
  let placeholderSrc = "";

  if (hash?.length > 0) {
    placeholderSrc = "/?"+hash;
  }

  onMount(() => {
    if (hash?.length > 0) {
      imageSrc =
        Env.S3_URL +
        "images/" +
        hash.substring(0, 12).replaceAll(".", "/").replaceAll("-", "=") +
        ".webp";
    } else {
      const sl = src.split(".")
      const ext = sl[sl.length - 1]
      imageSrc = src
      if(sl.length < 2 || !["jpeg","jpg","webp","avif","png"].includes(ext)){
        imageSrc += `-x${size}.avif`
      }
      imageSrc = Env.S3_URL + imageSrc
    }
	})
</script>

<div class={[s1.image_hash_ctn, css || ""].join(" ")}>
  {#if !!placeholderSrc}
    <img role={`0/0/${src}`} class="_bhi_" loading="lazy" alt="" /> 
  {/if}
  <img role={`1/${size}/${src}`} src={imageSrc} {alt} loading="lazy"
    onload={() => { placeholderSrc = "" }}
  />
</div>
