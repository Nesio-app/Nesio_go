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

    @objc func classifyImage(_ call: CAPPluginCall) {
        guard let base64Image = call.getString("base64Image"),
              let data = Data(base64Encoded: base64Image),
              let image = UIImage(data: data),
              let cgImage = image.cgImage else {
            call.reject("invalid image")
            return
        }

        let request = VNClassifyImageRequest { request, error in
            if let error = error {
                call.reject("vision classification failed: \(error.localizedDescription)")
                return
            }
            let observations = (request.results as? [VNClassificationObservation]) ?? []
            let labels = observations.prefix(5).map { "\($0.identifier) (\(Int($0.confidence * 100))%)" }
            call.resolve(["labels": labels])
        }

        let handler = VNImageRequestHandler(cgImage: cgImage, options: [:])
        do {
            try handler.perform([request])
        } catch {
            call.reject("vision classification request failed: \(error.localizedDescription)")
        }
    }

    @objc func inferSmartModel(_ call: CAPPluginCall) {
        guard let text = call.getString("text") else {
            call.reject("missing text")
            return
        }

        let normalized = text.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        if normalized.isEmpty {
            call.reject("missing text")
            return
        }

        let intent: String
        let summary: String
        let confidence: Double

        if normalized.contains("提醒") || normalized.contains("remind") || normalized.contains("todo") {
            intent = "reminder"
            summary = "这更像一个提醒或待办，需要补充时间和动作。"
            confidence = 0.87
        } else if normalized.contains("药") || normalized.contains("medicine") || normalized.contains("pill") {
            intent = "medicine"
            summary = "这更像药品或用药相关内容，可以继续提取剂量和频次。"
            confidence = 0.84
        } else if normalized.contains("票") || normalized.contains("invoice") || normalized.contains("receipt") {
            intent = "document"
            summary = "这更像票据或文档内容，可以做结构化整理。"
            confidence = 0.81
        } else if normalized.contains("人") || normalized.contains("person") {
            intent = "person"
            summary = "这更像人物相关内容，可以归档到联系人或关系节点。"
            confidence = 0.74
        } else {
            intent = "general"
            summary = "这是一个通用场景，适合继续补充上下文后再处理。"
            confidence = 0.62
        }

        call.resolve([
            "intent": intent,
            "summary": summary,
            "confidence": confidence,
            "input": text,
        ])
    }
}
