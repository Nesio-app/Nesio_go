import Foundation
import Capacitor
import HealthKit

@objc(HealthKitPlugin)
public class HealthKitPlugin: CAPPlugin {
    private let healthStore = HKHealthStore()

    @objc func requestPermission(_ call: CAPPluginCall) {
        guard HKHealthStore.isHealthDataAvailable() else {
            call.reject("health data not available")
            return
        }

        guard let stepsType = HKObjectType.quantityType(forIdentifier: .stepCount),
              let heartType = HKObjectType.quantityType(forIdentifier: .heartRate) else {
            call.reject("unable to load health types")
            return
        }

        let readTypes: Set<HKObjectType> = [stepsType, heartType]
        healthStore.requestAuthorization(toShare: [], read: readTypes) { success, error in
            if let error = error {
                call.reject("health permission failed: \(error.localizedDescription)")
                return
            }
            call.resolve(["granted": success])
        }
    }

    @objc func readTodaySteps(_ call: CAPPluginCall) {
        guard let stepsType = HKObjectType.quantityType(forIdentifier: .stepCount) else {
            call.reject("step type unavailable")
            return
        }

        let calendar = Calendar.current
        let startOfDay = calendar.startOfDay(for: Date())
        let predicate = HKQuery.predicateForSamples(withStart: startOfDay, end: Date(), options: .strictStartDate)
        let query = HKStatisticsQuery(quantityType: stepsType, quantitySamplePredicate: predicate, options: .cumulativeSum) { _, result, error in
            if let error = error {
                call.reject("query failed: \(error.localizedDescription)")
                return
            }
            let count = result?.sumQuantity()?.doubleValue(for: HKUnit.count()) ?? 0
            call.resolve(["steps": Int(count)])
        }
        healthStore.execute(query)
    }
}
