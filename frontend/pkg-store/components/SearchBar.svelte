<script lang="ts">
  const handleKeydown = (event: KeyboardEvent) => {
    const target = event.target as HTMLTextAreaElement;
    if (
      target &&
      (event.key === "Enter" || event.keyCode === 13 || event.charCode === 13)
    ) {
      // Prevent the default action (inserting a new line)
      event.preventDefault()
    }
  }
</script>

<div class="_1 relative lg:w-340 w-[34vw]">
  <textarea
    class="_2 w-full pl-12 rounded-[16px] pt-10 pl-14"
    cols="1"
    onkeydown={handleKeydown}
    placeholder="Buscar..."
  ></textarea>
  <i class="icon1-search absolute top-8 md:top-9 right-7 md:right-10"></i>
</div>

<style>

  ._1, ._2 {
    height: 36px;
  }
  ._1 > i {
    color: #6d698d;
  }
  ._2 {
    box-shadow:#20202329 0px 1px 3px, #20202329 0px 1px 2px;
    line-height: 1;
    outline: none;
    border: none;
    background-color: #eae9ef;
    resize: none;
  }

  @media (max-width: 740px) {
    ._1, ._2 {
      height: 36px;
      box-shadow: none;
    }
  }

</style>