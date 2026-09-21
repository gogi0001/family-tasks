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

        // Первый запуск — учитываем deep link из intent (холодный старт).
        val initialTaskId = extractTaskId(intent)
        val initial = serverUrlFromPrefs()
        if (initial.isEmpty()) {
            openSettings()
        } else {
            loadUrl(initial, initialTaskId)
        }
    }

    // singleTask: повторный тап по уведомлению, когда приложение уже живо.
    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)

        val taskId = extractTaskId(intent)
        Log.d("FamilyTasks", "onNewIntent, task = $taskId")

        if (taskId == null || loadedBase.isEmpty()) return

        // Меняем только hash — WebView не перезагружает страницу,
        // а браузер шлёт hashchange, который ловит наш JS.
        webView.loadUrl("$loadedBase/#task=${Uri.encode(taskId)}")
    }

    // Извлекаем параметр task из familytasks://open?task=<id>.
    private fun extractTaskId(intent: Intent?): String? {
        val data = intent?.data ?: return null
        if (data.scheme != "familytasks") return null
        return data.getQueryParameter("task")
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

    private fun loadUrl(url: String, taskId: String? = null) {
        val clean = url.trimEnd('/')
        loadedBase = clean
        val suffix = if (taskId != null) "#task=${Uri.encode(taskId)}" else ""
        webView.loadUrl("$clean/$suffix")
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