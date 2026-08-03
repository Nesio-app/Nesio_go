import Foundation
import Capacitor
import UserNotifications
import UIKit

@objc(PushNotification)
public class PushNotification: CAPPlugin, UNUserNotificationCenterDelegate {
    @objc func requestPermission(_ call: CAPPluginCall) {
        UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .badge, .sound]) { granted, error in
            if let error = error {
                call.reject("push permission failed: \(error.localizedDescription)")
                return
            }
            DispatchQueue.main.async {
                UIApplication.shared.registerForRemoteNotifications()
            }
            call.resolve(["granted": granted])
        }
    }

    @objc func getDeviceToken(_ call: CAPPluginCall) {
        let token = UserDefaults.standard.string(forKey: "nesio_push_token") ?? ""
        call.resolve(["token": token])
    }
}
