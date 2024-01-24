use image::{io::Reader as ImageReader, DynamicImage, GenericImageView};
use load_image::{self, export::rgb::{self}};
use webp;
// build for lambda arm: cargo build --target aarch64-unknown-linux-gnu --release
fn main() {
    //Get the current working directory into a string variable
    let current_dir = std::env::current_dir().unwrap().into_os_string().into_string().unwrap();
    println!("Iniciando. Current WD: {}", current_dir);

    let input_image = current_dir.clone() + "/test_inputs/demo_2.webp";

    //read an image in the current directory as an Image object
    let image_result = ImageReader::open(&input_image);
    if image_result.is_err(){
        println!("Error reading image: {}",image_result.err().unwrap().to_string());
        return;
    }
    // convert image_result as bytes into a new variable
    let img = image_result.unwrap().decode();
    if img.is_err(){   
        println!("Error decoding image: {}", img.err().unwrap().to_string());
        return;
    }

    let image_decoded = img.unwrap();
   
    let args = ConverArgs {
        image: image_decoded,
        widths: vec![240,500,900],
        directory: current_dir.clone() + "/test_outputs",
        name: "demo_img".to_string(),
    };

    convert_image(args);
}

struct ConverArgs {
    image: DynamicImage,
    widths: Vec<u32>,
    directory: String,
    name: String
}

fn convert_image(args: ConverArgs) {
    
    let mut dimensions: Vec<(u32, u32)> = Vec::new();

    for width in args.widths {
        let height = args.image.height() * width / args.image.width();
        dimensions.push((width, height));   
    }

    // Crea los archivos .webp
    for (width, height) in dimensions {
        println!("Conviertiendo imagen .webp en dimension: {}x{}...", width, height);
        // WEBP
        let image_resized = args.image.resize(width, height, image::imageops::FilterType::Triangle);
        let encoder: webp::Encoder = webp::Encoder::from_image(&image_resized).unwrap();

        let mut config = webp::WebPConfig::new().unwrap();
        config.lossless = 0;
        config.alpha_compression = 1;
        config.quality = 82.0;
        config.method = 6; // quality/speed trade-off (0=fast, 6=slower-better)

        // Encode the image at a specified quality 0-100
        let webp:  webp::WebPMemory = encoder.encode_advanced(&config).unwrap();
        let file_name_webp = format!("{}/{}-{}px.{}",args.directory, args.name, width, "webp");
        std::fs::write(&file_name_webp, &*webp).unwrap();

        println!("Imagen WEBP guardada en: {}", &file_name_webp);

        let mut image_vec1s: Vec<rgb::RGBA<u8>> = vec![];
        for i in 0..image_resized.height() { 
            for j in 0..image_resized.width() {
                let pixel = image_resized.get_pixel(j, i);
                let r = pixel[0];
                let g = pixel[1];
                let b = pixel[2];
                let a = pixel[3];
                image_vec1s.push(rgb::RGBA { r, g, b, a });
            }
        }

        println!("Conviertiendo imagen .avif en dimension: {}x{}...", width, height);
    
        // create a new variable of type imgref::ImgVec<RGBA8> and instanciate it with the above image
        let image_vec1 = imgref::ImgVec::new(image_vec1s, image_resized.width() as usize, image_resized.height() as usize);
        let image_img: imgref::Img<&[rgb::RGBA<u8>]> = imgref::Img::new(image_vec1.buf(), image_resized.dimensions().0 as usize, image_resized.dimensions().1 as usize);
    
        let res = ravif::Encoder::new()
                .with_quality(80.)
                .with_speed(2)
                .encode_rgba( image_img);
    
        if res.is_err(){
            println!("Error encoding image: {}", res.err().unwrap().to_string());
            return;
        }

        let file_name_avif = format!("{}/{}-{}px.{}",args.directory, args.name, width, "avif");
                
        let result_saved = std::fs::write(&file_name_avif, res.unwrap().avif_file);
        if result_saved.is_err() {
            println!("Error saving image: {}", result_saved.err().unwrap().to_string());
            return; 
        }

        println!("Imagen AVIF guardada en: {}", &file_name_avif);
    }   
}