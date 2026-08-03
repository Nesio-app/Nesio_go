import Foundation
import Capacitor
import CoreLocation

@objc(LocationPlugin)
public class LocationPlugin: CAPPlugin, CLLocationManagerDelegate {
    private let manager = CLLocationManager()
    private var currentCall: CAPPluginCall?

    public override func load() {
        manager.delegate = self
        manager.desiredAccuracy = kCLLocationAccuracyBest
    }

    @objc func requestPermission(_ call: CAPPluginCall) {
        currentCall = call
        manager.requestWhenInUseAuthorization()
    }

    @objc func currentLocation(_ call: CAPPluginCall) {
        currentCall = call
        let status = manager.authorizationStatus
        if status == .notDetermined {
            manager.requestWhenInUseAuthorization()
            return
        }
        if status == .denied || status == .restricted {
            call.reject("location permission denied")
            return
        }
        manager.requestLocation()
    }

    public func locationManager(_ manager: CLLocationManager, didChangeAuthorization status: CLAuthorizationStatus) {
        if status == .authorizedAlways || status == .authorizedWhenInUse {
            currentCall?.resolve(["granted": true])
            currentCall = nil
        } else if status == .denied || status == .restricted {
            currentCall?.resolve(["granted": false])
            currentCall = nil
        }
    }

    public func locationManager(_ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]) {
        guard let location = locations.first else {
            currentCall?.reject("no location")
            currentCall = nil
            return
        }
        currentCall?.resolve([
            "latitude": location.coordinate.latitude,
            "longitude": location.coordinate.longitude,
            "accuracy": location.horizontalAccuracy
        ])
        currentCall = nil
    }

    public func locationManager(_ manager: CLLocationManager, didFailWithError error: Error) {
        currentCall?.reject("location failed: \(error.localizedDescription)")
        currentCall = nil
    }
}
