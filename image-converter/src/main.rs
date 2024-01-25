use image::{io::Reader as ImageReader, DynamicImage, GenericImageView};
use load_image::{self, export::rgb::{self}};
use base64::prelude::*;
use webp;
// build for lambda arm: cargo build --target aarch64-unknown-linux-gnu --release
struct ConverArgs {
    // create mutable property image
    image: DynamicImage,
    resolutions: Vec<u32>,
    output_directory: String,
    name: String,
    webp_quality: u32,
    webp_method: i32,
    avif_quality: u32,
    avif_speed: u8,
    use_webp: bool,
    use_avif: bool,
}
fn main() {
    let mut convert_args = ConverArgs {
        image: DynamicImage::new_rgba8(0, 0),
        resolutions: vec![],
        output_directory: "".to_string(),
        name: "".to_string(),
        webp_quality: 82,
        webp_method: 6,
        avif_quality: 80,
        avif_speed: 2,
        use_webp: false,
        use_avif: false,
    };

    //get execution arguments in a variable
    let args: Vec<String> = std::env::args().collect();
    let current_dir = std::env::current_dir().unwrap().into_os_string().into_string().unwrap();
    convert_args.output_directory = current_dir.clone();

    let mut first_arg: String = "".to_owned();
    for (_, _arg) in args.iter().enumerate(){
        let arg = _arg.trim();

        if arg.len() > 7 && &arg[0..7] == "-image=" {
            first_arg = arg.to_string();
            break;
        }
        if &arg[0..1] == "-" {
            break;
        } else {
            first_arg = arg.to_string();
        }
    }

    if first_arg.len() > 7 && &first_arg[0..7] == "-image=" {
        let base64_image = &first_arg[7..];
        let image_data = BASE64_STANDARD.decode(base64_image).unwrap();
        // read the image data as a DynamicImage object
        let image = image::load_from_memory(&image_data);
        if image.is_err(){
            println!("Error: reading image: {}",image.err().unwrap().to_string());
            return;
        }
        convert_args.image = image.unwrap();
    } else {
        let image_path: String;
        if &first_arg[0..1] == "-" {
            println!("Error: must provide an image (name, path or base64) in the first argument");
            return;
        }
        if &first_arg[0..1] == "/" {
            image_path = first_arg;
        } else if &first_arg[0..2] == "./" {
            image_path = current_dir.clone() + &first_arg[1..];
        } else {
            image_path = current_dir.clone() + "/" + &first_arg;
        }
        let image_paths: Vec<&str> = image_path.split("/").collect();
        convert_args.name = image_paths[image_paths.len()-1].to_string();
        
        println!("Reading image: {}",image_path);

        let image_result = ImageReader::open(&image_path);
        if image_result.is_err(){
            println!("Error: reading image: {}",image_result.err().unwrap().to_string());
            return;
        }
        let image = image_result.unwrap().decode();
        if image.is_err(){
            println!("Error: reading image: {}",image.err().unwrap().to_string());
            return;
        }
        convert_args.image = image.unwrap();
    }

    for (_, arg) in args.iter().enumerate(){
        if arg == "-webp" {
            convert_args.use_webp = true
        } else if arg == "-avif" {
            convert_args.use_avif = true
        } else if arg.contains("=") {
            let args_equals: Vec<&str> = arg.split("=").collect();
            let arg_name = args_equals[0];
            let arg_value = args_equals[1];
            if arg_name == "-webp-quality" {
                convert_args.use_webp = true;
                convert_args.webp_quality = arg_value.parse::<u32>().unwrap();
            } else if arg_name == "-webp-method" {
                convert_args.use_avif = true;
                convert_args.webp_method = arg_value.parse::<i32>().unwrap();
            } else if arg_name == "-avif-quality" {
                convert_args.use_avif = true;
                convert_args.avif_quality = arg_value.parse::<u32>().unwrap();
            } else if arg_name == "-avif-speed" {
                convert_args.use_avif = true;
                convert_args.avif_speed = arg_value.parse::<u8>().unwrap();
            } else if arg_name == "-output" {
                convert_args.output_directory = arg_value.to_string();
            } else if arg_name == "-resolutions" {
                let resolutions: Vec<&str> = arg_value.split(",").collect();
                for resolution in resolutions {
                    let vu32 = resolution.parse::<u32>();
                    if vu32.is_err() {
                        println!("Error: Invalid resolution format: {}", vu32.unwrap_err().to_string());
                        return;
                    }
                    let resolution = vu32.unwrap();
                    if resolution > 4000 {
                        println!("Error: Invalid resolution: {} | Too Big (>16 mpx)", resolution);
                        return;
                    }
                    convert_args.resolutions.push(resolution);
                }
            }
        }
    }
    convert_image(convert_args);
}

fn convert_image(args: ConverArgs) {
    
    let mut dimensions: Vec<(u32, u32)> = Vec::new();

    for width in args.resolutions {
        let height = args.image.height() * width / args.image.width();
        dimensions.push((width, height));   
    }

    // Crea los archivos .webp
    for (width, height) in dimensions {
        let image_resized = args.image.resize(
            width, height, image::imageops::FilterType::Triangle);
        if args.use_webp { // WEBP
            println!("Conviertiendo imagen .webp en dimension: {}x{}...", width, height);
            let encoder: webp::Encoder = webp::Encoder::from_image(&image_resized).unwrap();
    
            let mut config = webp::WebPConfig::new().unwrap();
            config.lossless = 0;
            config.alpha_compression = 1;
            config.quality = 82.0;
            config.method = 6; // quality/speed trade-off (0=fast, 6=slower-better)
    
            // Encode the image at a specified quality 0-100
            let webp:  webp::WebPMemory = encoder.encode_advanced(&config).unwrap();
            let file_name_webp = format!("{}/{}-{}px.{}",args.output_directory, args.name, width, "webp");
            std::fs::write(&file_name_webp, &*webp).unwrap();
    
            println!("Imagen WEBP guardada en: {}", &file_name_webp);
        }
        if args.use_avif { // AVIF
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
    
            let file_name_avif = format!("{}/{}-{}px.{}",args.output_directory, args.name, width, "avif");
                    
            let result_saved = std::fs::write(&file_name_avif, res.unwrap().avif_file);
            if result_saved.is_err() {
                println!("Error saving image: {}", result_saved.err().unwrap().to_string());
                return; 
            }
    
            println!("Imagen AVIF guardada en: {}", &file_name_avif);
        }
    }
}