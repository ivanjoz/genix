<script lang="ts">
import ImageUploader from '$ui/components/ImageUploader.svelte';
import Page from '$ui/components/Page.svelte';
    import { DemoService } from "./demo-service.svelte";

	let demoService = new DemoService()

</script>

<Page title="demo">
	<div>hola</div>
	<div>{demoService.message}</div>
	<ImageUploader saveAPI="producto-image"
		clearOnUpload={true}
		cardCss="w-200 h-200 p-4"
		setDataToSend={e => {
			e.ProductoID = 8
		}}
		onUploaded={src => {
			console.log("imagen subida::", src)
		}}
	/>
</Page>

<style>

</style>
