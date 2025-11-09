<script lang="ts">
    import ImageUploader from "../../../components/ImageUploader.svelte";
	import Page from "../../../components/Page.svelte";
    import { DemoService } from "./demo-service.svelte";

	let demoService = new DemoService()

</script>

<Page>
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

