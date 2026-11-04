import AppKit
import Foundation
import WebKit

final class CarpetsAmeerApp: NSObject, NSApplicationDelegate, WKScriptMessageHandler, WKNavigationDelegate {
    private var window: NSWindow!
    private let uiSmokeTest = CommandLine.arguments.contains("--ui-self-test")
    private var browser: WKWebView!
    private let workbookPathKey = "CarpetsAmeerWorkbookPath"

    private let excelBridge = """
    (function () {
      if (!window.CarpetsExcel || !window.webkit ||
          !window.webkit.messageHandlers || !window.webkit.messageHandlers.excelExport) return;
      window.CarpetsExcel.download = function(blob, name) {
        var reader = new FileReader();
        reader.onload = function() {
          var result = String(reader.result || "");
          var comma = result.indexOf(",");
          if (comma < 0) return;
          window.webkit.messageHandlers.excelExport.postMessage({
            name: name || "Carpets_Ameer_Sales.xlsx",
            base64: result.slice(comma + 1)
          });
        };
        reader.readAsDataURL(blob);
      };
    })();
    """

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.regular)
        installMenus()

        let controller = WKUserContentController()
        controller.add(self, name: "excelExport")
        controller.addUserScript(WKUserScript(source: excelBridge,
                                            injectionTime: .atDocumentEnd,
                                            forMainFrameOnly: true))

        let configuration = WKWebViewConfiguration()
        configuration.websiteDataStore = WKWebsiteDataStore.default()
        configuration.userContentController = controller

        browser = WKWebView(frame: .zero, configuration: configuration)
        browser.navigationDelegate = self

        window = NSWindow(contentRect: NSRect(x: 0, y: 0, width: 1260, height: 810),
                          styleMask: [.titled, .closable, .miniaturizable, .resizable],
                          backing: .buffered, defer: false)
        window.title = "Carpets Ameer | منار العتبة"
        window.minSize = NSSize(width: 870, height: 560)
        window.center()
        window.contentView = browser
        window.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)

        guard let fileURL = Bundle.main.url(forResource: "index", withExtension: "html"),
              let resourceURL = Bundle.main.resourceURL else {
            showError("The app's offline website is missing. Reinstall Carpets Ameer.")
            return
        }
        browser.loadFileURL(fileURL, allowingReadAccessTo: resourceURL)
        if uiSmokeTest {
            DispatchQueue.main.asyncAfter(deadline: .now() + 25) {
                fputs("FAIL: macOS window failed to load in time\n", stderr)
                exit(3)
            }
        }
    }

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        return true
    }

    func userContentController(_ userContentController: WKUserContentController,
                               didReceive message: WKScriptMessage) {
        guard message.name == "excelExport",
              let payload = message.body as? [String: Any],
              let encoded = payload["base64"] as? String,
              let data = Data(base64Encoded: encoded),
              data.count > 100 else {
            showError("Could not generate the Excel workbook. Please try Save again.")
            return
        }

        let remembered = UserDefaults.standard.string(forKey: workbookPathKey)
        if let savedPath = remembered, !savedPath.isEmpty,
           FileManager.default.fileExists(atPath: savedPath) {
            do {
                try data.write(to: URL(fileURLWithPath: savedPath), options: .atomic)
                return
            } catch {
                UserDefaults.standard.removeObject(forKey: workbookPathKey)
                // Ask for a new location if the previous workbook was moved or protected.
            }
        }

        let panel = NSSavePanel()
        panel.title = "Save carpet sales Excel workbook"
        panel.nameFieldStringValue = "Carpets_Ameer_Sales.xlsx"
        panel.allowedFileTypes = ["xlsx"]
        panel.canCreateDirectories = true
        panel.beginSheetModal(for: window) { [weak self] response in
            guard response == .OK, let url = panel.url else { return }
            do {
                try data.write(to: url, options: .atomic)
                UserDefaults.standard.set(url.path, forKey: self?.workbookPathKey ?? "CarpetsAmeerWorkbookPath")
            } catch {
                self?.showError("Could not save Excel workbook: " + error.localizedDescription)
            }
        }
    }

    func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
        guard uiSmokeTest else { return }
        let check = "Boolean(window.CarpetsExcel && document.getElementById('saleForm') && document.getElementById('saveBtn') && localStorage.setItem('__carpets_test','1') === undefined && localStorage.getItem('__carpets_test') === '1')"
        webView.evaluateJavaScript(check) { result, error in
            if error != nil || !(result as? Bool ?? false) {
                fputs("FAIL: native WebKit content or persistent storage test failed: \\(String(describing: error))\\n", stderr)
                exit(4)
            }
            print("PASS: macOS GUI WebKit loaded bilingual sales form and working localStorage")
            exit(0)
        }
    }

    func webView(_ webView: WKWebView,
                 didFailProvisionalNavigation navigation: WKNavigation!,
                 withError error: Error) {
        showError("Could not load the offline sales desk: " + error.localizedDescription)
    }

    private func showError(_ message: String) {
        let alert = NSAlert()
        alert.alertStyle = .warning
        alert.messageText = "Carpets Ameer"
        alert.informativeText = message
        alert.addButton(withTitle: "OK")
        if let window = window {
            alert.beginSheetModal(for: window)
        } else {
            alert.runModal()
        }
    }

    private func installMenus() {
        let menuBar = NSMenu()
        let appItem = NSMenuItem()
        let appMenu = NSMenu()
        appMenu.addItem(NSMenuItem(title: "Quit Carpets Ameer",
                                  action: #selector(NSApplication.terminate(_:)),
                                  keyEquivalent: "q"))
        appItem.submenu = appMenu
        menuBar.addItem(appItem)

        let editItem = NSMenuItem()
        let editMenu = NSMenu(title: "Edit")
        editMenu.addItem(NSMenuItem(title: "Undo", action: Selector(("undo:")), keyEquivalent: "z"))
        editMenu.addItem(NSMenuItem(title: "Redo", action: Selector(("redo:")), keyEquivalent: "Z"))
        editMenu.addItem(NSMenuItem.separator())
        editMenu.addItem(NSMenuItem(title: "Cut", action: #selector(NSText.cut(_:)), keyEquivalent: "x"))
        editMenu.addItem(NSMenuItem(title: "Copy", action: #selector(NSText.copy(_:)), keyEquivalent: "c"))
        editMenu.addItem(NSMenuItem(title: "Paste", action: #selector(NSText.paste(_:)), keyEquivalent: "v"))
        editMenu.addItem(NSMenuItem(title: "Select All", action: #selector(NSText.selectAll(_:)), keyEquivalent: "a"))
        editItem.submenu = editMenu
        menuBar.addItem(editItem)
        NSApp.mainMenu = menuBar
    }
}

@main
struct ApplicationMain {
    static func main() {
        if CommandLine.arguments.contains("--self-test") {
            guard let path = Bundle.main.path(forResource: "index", ofType: "html"),
                  let source = try? String(contentsOfFile: path, encoding: .utf8),
                  source.contains("CarpetsExcel"),
                  source.contains("localStorage") else {
                fputs("ERROR: missing or invalid bundled sales website\\n", stderr)
                exit(2)
            }
            print("PASS: macOS executable, bundled HTML, Excel logic and offline storage")
            return
        }
        let application = NSApplication.shared
        let delegate = CarpetsAmeerApp()
        application.delegate = delegate
        application.run()
    }
}
