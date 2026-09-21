package com.gogi0001.familytasks

import android.annotation.SuppressLint
import android.app.Activity
import android.content.Intent
import android.net.Uri
import android.os.Bundle
import android.util.Log
import android.view.Menu
import android.view.MenuItem
import android.view.View
import android.webkit.CookieManager
import android.webkit.WebChromeClient
import android.webkit.WebResourceRequest
import android.webkit.WebSettings
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity

class MainActivity : AppCompatActivity() {

    private lateinit var webView: WebView
    private lateinit var progress: View

    // URL, с которого сейчас загружена страница (без завершающего слэша).
    private var loadedBase: String = ""

    private val settingsLauncher = registerForActivityResult(
        ActivityResultContracts.StartActivityForResult()
    ) { result ->
        if (result.resultCode == Activity.RESULT_OK) {
            val newUrl = result.data?.getStringExtra("serverUrl")
                ?.trimEnd('/')
                .orEmpty()
            if (newUrl.isNotEmpty() && newUrl != loadedBase) {
                loadUrl(newUrl)
            }
        }
    }

    @SuppressLint("SetJavaScriptEnabled")
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        webView = findViewById(R.id.webview)
        progress = findViewById(R.id.progress)

        CookieManager.getInstance().setAcceptCookie(true)
        CookieManager.getInstance().setAcceptThirdPartyCookies(webView, false)

        webView.settings.apply {
            javaScriptEnabled = true
            domStorageEnabled = true
            databaseEnabled = true
            cacheMode = WebSettings.LOAD_DEFAULT
            loadWithOverviewMode = true
            useWideViewPort = true
            mediaPlaybackRequiresUserGesture = false
        }

        webView.webViewClient = object : WebViewClient() {
            override fun shouldOverrideUrlLoading(
                view: WebView,
                request: WebResourceRequest
            ): Boolean {
                val url = request.url.toString()

                if (loadedBase.isEmpty()) return false

                if (!url.startsWith("$loadedBase/") && url != loadedBase) {
                    startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
                    return true
                }
                return false
            }
        }

        webView.webChromeClient = object : WebChromeClient() {
            override fun onProgressChanged(view: WebView, newProgress: Int) {
                progress.visibility = if (newProgress in 1..99) View.VISIBLE else View.GONE
            }
        }

        // Первый запуск — берём URL из prefs или уходим в настройки.
        val initial = serverUrlFromPrefs()
        if (initial.isEmpty()) {
            openSettings()
        } else {
            loadUrl(initial)
        }

        // Если приложение стартовало из уведомления — обработаем intent.
        handleDeepLink(intent)
    }

    // singleTask + deep link: повторный тап по уведомлению приходит сюда.
    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        handleDeepLink(intent)
    }

    // Разбор familytasks://... из уведомления.
    private fun handleDeepLink(intent: Intent?) {
        val data = intent?.data ?: return
        if (data.scheme != "familytasks") return

        Log.d("FamilyTasks", "deep link: $data")

        // Приложение уже открыто — достаточно перезагрузить WebView,
        // чтобы увидеть свежие данные. Параметр task (если появится
        // в будущем) можно использовать для открытия конкретной задачи.
        if (loadedBase.isNotEmpty()) {
            webView.reload()
        }
    }

    override fun onResume() {
        super.onResume()
        val fromPrefs = serverUrlFromPrefs()
        if (fromPrefs.isNotEmpty() && fromPrefs != loadedBase) {
            loadUrl(fromPrefs)
        } else if (loadedBase.isEmpty() && fromPrefs.isEmpty()) {
            openSettings()
        }
    }

    override fun onPause() {
        super.onPause()
        CookieManager.getInstance().flush()
    }

    private fun loadUrl(url: String) {
        val clean = url.trimEnd('/')
        loadedBase = clean
        webView.loadUrl("$clean/")
    }

    private fun openSettings() {
        settingsLauncher.launch(Intent(this, SettingsActivity::class.java))
    }

    private fun serverUrlFromPrefs(): String {
        val prefs = getSharedPreferences("app", MODE_PRIVATE)
        return prefs.getString("serverUrl", "").orEmpty().trim().trimEnd('/')
    }

    override fun onCreateOptionsMenu(menu: Menu): Boolean {
        menuInflater.inflate(R.menu.main, menu)
        return true
    }

    override fun onOptionsItemSelected(item: MenuItem): Boolean {
        return when (item.itemId) {
            R.id.action_reload -> {
                if (loadedBase.isNotEmpty()) webView.reload()
                true
            }
            R.id.action_settings -> {
                openSettings()
                true
            }
            else -> super.onOptionsItemSelected(item)
        }
    }

    @Deprecated("Deprecated in Java")
    override fun onBackPressed() {
        if (webView.canGoBack()) webView.goBack()
        else super.onBackPressed()
    }
}