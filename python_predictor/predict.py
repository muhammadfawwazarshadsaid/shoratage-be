import sys
import json
import os
from ultralytics import YOLO
from PIL import Image

def run_prediction(image_path):
    """
    Menjalankan prediksi pada satu gambar dan mencetak hasilnya sebagai JSON.
    """
    try:
        script_dir = os.path.dirname(__file__)
        model_path = os.path.join(script_dir, 'best.pt')
        
        if not os.path.exists(model_path):
            raise FileNotFoundError(f"Model file not found at {model_path}")

        model = YOLO(model_path)
        img = Image.open(image_path)
        results = model(img)

        detections = []
        for r in results:
            for box in r.boxes:
                x1, y1, x2, y2 = map(int, box.xyxy[0].tolist())
                confidence = round(float(box.conf[0]), 4)
                class_id = int(box.cls[0])
                class_name = model.names[class_id]

                detections.append({
                    'class_name': class_name,
                    'confidence': confidence,
                    'box': [x1, y1, x2, y2]
                })
        
        print(json.dumps(detections))

    except Exception as e:
        error_output = {"error": str(e)}
        print(json.dumps(error_output), file=sys.stderr)
        sys.exit(1)

if __name__ == '__main__':
    if len(sys.argv) != 2:
        print(json.dumps({"error": "Usage: python predict.py <image_path>"}), file=sys.stderr)
        sys.exit(1)
    
    image_file_path = sys.argv[1]
    run_prediction(image_file_path)