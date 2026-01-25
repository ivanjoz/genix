<script>
import Input from '$ui/components/Input.svelte';
import SearchSelect from '$ui/components/SearchSelect.svelte';
   
  let options = [
    { id: 1, nombre: "Ivan Joseph Angulo" },
    { id: 2, nombre: "Pedro Pascal Armando" },
    { id: 3, nombre: "Sergio Bonamasa" },
    { id: 4, nombre: "Ricardo Ríos del Agua" },
  ]

  let form = {};
</script>

<div class="p-12 flex">
  <div class="mr-20">
    <div class="h-40">
      <div class="text-[24px]">Componentes</div>
    </div>
    <Input label="Nombre Usuario das das dasd asd asdas" css="w-200 m-6" saveOn={form} save="nombre" />

    <Input label="Nombre Usuario 2" css="w-200 m-6" saveOn={form} save="nombre" />

    <SearchSelect label="Usuarios" css="w-200 mr-6" saveOn={form} save="ciudadID"
      options={options} keys="id.nombre"
    />

    <button
      onclick={() => {
        console.log(form);
      }}>Enviar</button
    >
  </div>
  <div class="_1 p-16">
    <div class="h-40">
      <div class="text-[24px]">Componentes</div>
    </div>
    <Input label="Nombre Usuario das das dasd asd asdas" css="w-200 m-6" saveOn={form} save="nombre"></Input>

    <Input label="Nombre Usuario 2" css="w-200 m-6" saveOn={form} save="nombre"></Input>

    <button
      onclick={() => {
        console.log(form);
      }}>Enviar</button
    >
  </div>
</div>

<style>
  ._1 {
    background-color: #e9eef1;
  }
</style>