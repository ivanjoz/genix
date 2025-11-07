<script lang="ts">
	import Page from "../../../components/Page.svelte";
    import { DemoService } from "./demo-service.svelte";

	let demoService = new DemoService()

</script>

<Page>
	<div>hola</div>
	<div>{demoService.message}</div>
</Page>

<style>

</style>

