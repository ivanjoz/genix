<script lang="ts">
import ImageUploader from '$components/files/ImageUploader.svelte';
import Page from '$domain/Page.svelte';
    import { DemoService } from "./demo-service.svelte";

	let demoService = new DemoService()

</script>

<Page title="demo">
	<div>hola</div>
	<div>{demoService.message}</div>
	<ImageUploader saveAPI="product-image"
		clearOnUpload={true}
		cardCss="w-200 h-200 p-4"
		setDataToSend={e => {
			e.ProductID = 8
		}}
		onUploaded={image => {
			console.log("imagen subida::", image)
		}}
	/>
</Page>

<style>

</style>
