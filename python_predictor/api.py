import io
import os
import base64
import cv2
import numpy as np
import tempfile
from flask import Flask, request, jsonify
from flask_cors import CORS
from ultralytics import YOLO
from PIL import Image

app = Flask(__name__)
CORS(app)

try:
    script_dir = os.path.dirname(__file__)
    default_model_path = os.path.join(script_dir, 'best.pt')
    default_model = YOLO(default_model_path)
    print("✅ Model default 'best.pt' berhasil dimuat.")
except Exception as e:
    print(f"❌ Error saat memuat model default: {e}")
    default_model = None

@app.route('/predict', methods=['POST'])
def predict():
    active_model = default_model
    temp_model_file = None

    if 'model' in request.files:
        model_file = request.files['model']
        if model_file.filename != '':
            try:
                with tempfile.NamedTemporaryFile(suffix=".pt", delete=False) as temp:
                    model_file.save(temp.name)
                    temp_model_file = temp.name
                print(f"Menggunakan model custom: {model_file.filename}")
                active_model = YOLO(temp_model_file)
            except Exception as e:
                return jsonify({'error': f'Gagal memuat model custom: {str(e)}'}), 500

    if active_model is None:
        return jsonify({'error': 'Model tidak berhasil dimuat'}), 500

    if 'file' not in request.files:
        return jsonify({'error': 'File gambar tidak ditemukan'}), 400

    image_file = request.files['file']
    if not image_file:
        return jsonify({'error': 'File gambar tidak valid'}), 400

    try:
        conf_threshold = float(request.form.get('conf', 0.25))
        iou_threshold = float(request.form.get('iou', 0.7))
        agnostic_nms = request.form.get('agnostic_nms', 'false').lower() == 'true'
        
        print(f"Konfigurasi: conf={conf_threshold}, iou={iou_threshold}, agnostic_nms={agnostic_nms}")
    except ValueError:
        return jsonify({'error': 'Nilai conf atau iou tidak valid'}), 400

    try:
        img_bytes = image_file.read()
        img = Image.open(io.BytesIO(img_bytes))
        
        results = active_model(
            img,
            conf=conf_threshold,
            iou=iou_threshold,
            agnostic_nms=agnostic_nms
        )

        summary_data = {}
        for r in results:
            for box in r.boxes:
                class_id = int(box.cls[0])
                class_name = active_model.names[class_id]
                confidence = round(float(box.conf[0]), 4)
                if class_name not in summary_data:
                    summary_data[class_name] = {'count': 0, 'total_confidence': 0.0}
                summary_data[class_name]['count'] += 1
                summary_data[class_name]['total_confidence'] += confidence
        
        detection_summary = []
        for name, data in summary_data.items():
            avg_confidence = data['total_confidence'] / data['count']
            detection_summary.append({
                'class_name': name, 'quantity': data['count'], 'avg_confidence': round(avg_confidence, 4)
            })

        annotated_image_rgb = results[0].plot()
        annotated_image_bgr = cv2.cvtColor(annotated_image_rgb, cv2.COLOR_RGB2BGR)
        _, buffer = cv2.imencode('.jpg', annotated_image_bgr)
        base64_image = base64.b64encode(buffer).decode('utf-8')
        data_url = f"data:image/jpeg;base64,{base64_image}"

        return jsonify({"summary": detection_summary, "annotated_image": data_url})

    except Exception as e:
        return jsonify({'error': f'Gagal memproses gambar: {str(e)}'}), 500
    finally:
        if temp_model_file:
            os.remove(temp_model_file)
            print(f"File model sementara '{temp_model_file}' dihapus.")

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5001, debug=True)