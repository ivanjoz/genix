<script lang="ts">
import Page from '$domain/Page.svelte';
    import { DemoService } from "./demo-service.svelte";

	let demoService = new DemoService()

</script>

<Page title="demo">
	<div>hola</div>
	<div>{demoService.message}</div>
</Page>

<style>

</style>
