import Foundation
import Capacitor
import Vision
import UIKit

@objc(VisionPlugin)
public class VisionPlugin: CAPPlugin {
    @objc func recognizeText(_ call: CAPPluginCall) {
        guard let base64Image = call.getString("base64Image"),
              let data = Data(base64Encoded: base64Image),
              let image = UIImage(data: data),
              let cgImage = image.cgImage else {
            call.reject("invalid image")
            return
        }

        let request = VNRecognizeTextRequest { request, error in
            if let error = error {
                call.reject("vision failed: \(error.localizedDescription)")
                return
            }
            let observations = (request.results as? [VNRecognizedTextObservation]) ?? []
            let lines = observations.compactMap { $0.topCandidates(1).first?.string }
            call.resolve(["text": lines.joined(separator: "\n"), "lines": lines])
        }
        request.recognitionLevel = .accurate
        request.usesLanguageCorrection = true

        let handler = VNImageRequestHandler(cgImage: cgImage, options: [:])
        do {
            try handler.perform([request])
        } catch {
            call.reject("vision request failed: \(error.localizedDescription)")
        }
    }
}
