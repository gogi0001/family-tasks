package com.gogi0001.familytasks

import android.annotation.SuppressLint
import android.app.Activity
import android.content.ActivityNotFoundException
import android.content.Intent
import android.net.Uri
import android.os.Bundle
import android.util.Log
import android.view.Menu
import android.view.MenuItem
import android.view.View
import android.webkit.CookieManager
import android.webkit.ValueCallback
import android.webkit.WebChromeClient
import android.webkit.WebResourceRequest
import android.webkit.WebSettings
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.FileProvider
import com.google.android.material.appbar.MaterialToolbar
import java.io.File

class MainActivity : AppCompatActivity() {

    private lateinit var webView: WebView
    private lateinit var progress: View

    private var loadedBase: String = ""

    // Колбэк, который WebView ждёт, пока пользователь выбирает файл.
    private var filePathCallback: ValueCallback<Array<Uri>>? = null

    // URI временного файла, куда камера запишет снимок.
    private var cameraOutputUri: Uri? = null

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

    private val fileChooserLauncher = registerForActivityResult(
        ActivityResultContracts.StartActivityForResult()
    ) { result ->
        val callback = filePathCallback
        filePathCallback = null

        if (callback == null) return@registerForActivityResult

        // Отмена
        if (result.resultCode != Activity.RESULT_OK) {
            // Если использовалась камера — удаляем временный файл.
            cameraOutputUri?.let { uri ->
                deleteCacheFile(uri)
            }
            cameraOutputUri = null
            callback.onReceiveValue(null)
            return@registerForActivityResult
        }

        val data = result.data

        // Случай А: пользователь выбрал файл из галереи / файлового менеджера.
        val fromGallery: Array<Uri>? = when {
            data?.clipData != null -> {
                val cd = data.clipData!!
                Array(cd.itemCount) { i -> cd.getItemAt(i).uri }
            }
            data?.data != null -> arrayOf(data.data!!)
            else -> null
        }

        // Случай Б: пользователь сделал снимок через камеру.
        // В этом случае data == null (или data.data == null), а результат лежит
        // в cameraOutputUri, который мы передали камере через EXTRA_OUTPUT.
        if (fromGallery == null && cameraOutputUri != null) {
            callback.onReceiveValue(arrayOf(cameraOutputUri!!))
            cameraOutputUri = null
            return@registerForActivityResult
        }

        cameraOutputUri = null
        callback.onReceiveValue(fromGallery)
    }

    @SuppressLint("SetJavaScriptEnabled")
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        val toolbar = findViewById<MaterialToolbar>(R.id.toolbar)
        setSupportActionBar(toolbar)

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
            allowFileAccess = true
            allowContentAccess = true
            saveFormData = false
            if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.O) {
                safeBrowsingEnabled = false
            }
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

            override fun onShowFileChooser(
                webView: WebView,
                filePathCallbackLocal: ValueCallback<Array<Uri>>,
                fileChooserParams: FileChooserParams
            ): Boolean {
                // Отменяем предыдущий «висящий» запрос, если он был.
                filePathCallback?.onReceiveValue(null)
                filePathCallback = filePathCallbackLocal
                cameraOutputUri = null

                // Основной интент — стандартный системный file picker.
                val contentIntent = fileChooserParams.createIntent().apply {
                    addCategory(Intent.CATEGORY_OPENABLE)
                }

                // Дополнительный интент — камера.
                val cameraIntent = buildCameraIntent()

                // Собираем chooser: file picker + камера.
                val chooser = Intent.createChooser(contentIntent, "Выбор файла").apply {
                    if (cameraIntent != null) {
                        putExtra(Intent.EXTRA_INITIAL_INTENTS, arrayOf(cameraIntent))
                    }
                }

                return try {
                    fileChooserLauncher.launch(chooser)
                    true
                } catch (e: ActivityNotFoundException) {
                    Log.e("FamilyTasks", "no activity for file chooser", e)
                    filePathCallback = null
                    filePathCallbackLocal.onReceiveValue(null)
                    cameraOutputUri = null
                    false
                }
            }
        }

        val initialTaskId = extractTaskId(intent)
        val initial = serverUrlFromPrefs()
        if (initial.isEmpty()) {
            openSettings()
        } else {
            loadUrl(initial, initialTaskId)
        }
    }

    // Готовит Intent для камеры, попутно создавая файл для снимка
    // и получая на него content:// URI через FileProvider.
    private fun buildCameraIntent(): Intent? {
        val cameraIntent = Intent(android.provider.MediaStore.ACTION_IMAGE_CAPTURE)

        // Проверяем наличие хотя бы одного обработчика через queryIntentActivities
        // (это работает и на API 30+, если в манифесте есть <queries>).
        val handlers = packageManager.queryIntentActivities(
            cameraIntent,
            android.content.pm.PackageManager.MATCH_DEFAULT_ONLY
        )
        Log.d("FamilyTasks", "camera handlers: ${handlers.size}")
        if (handlers.isEmpty()) {
            return null
        }

        return try {
            val dir = File(cacheDir, "camera").apply { mkdirs() }
            val photoFile = File.createTempFile("capture_", ".jpg", dir)

            val uri = FileProvider.getUriForFile(
                this,
                "${packageName}.fileprovider",
                photoFile
            )
            cameraOutputUri = uri
            Log.d("FamilyTasks", "camera output URI: $uri")

            cameraIntent.apply {
                putExtra(android.provider.MediaStore.EXTRA_OUTPUT, uri)
                addFlags(Intent.FLAG_GRANT_WRITE_URI_PERMISSION)
                addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
            }
        } catch (e: Exception) {
            Log.e("FamilyTasks", "camera intent failed", e)
            cameraOutputUri = null
            null
        }
    }
    // Удаляет временный файл снимка, если пользователь отменил выбор.
    private fun deleteCacheFile(uri: Uri) {
        try {
            val path = uri.path ?: return
            // content://<authority>/<encoded-path>
            val file = File(cacheDir, "camera/" + File(path).name)
            if (file.exists()) file.delete()
        } catch (_: Exception) {
            // Не критично — временный файл перезапишется в следующий раз.
        }
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)

        val taskId = extractTaskId(intent)
        Log.d("FamilyTasks", "onNewIntent, task = $taskId")

        if (taskId == null || loadedBase.isEmpty()) return
        webView.loadUrl("$loadedBase/#task=${Uri.encode(taskId)}")
    }

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

    override fun onSaveInstanceState(outState: Bundle) {
        super.onSaveInstanceState(outState)
        filePathCallback?.onReceiveValue(null)
        filePathCallback = null
        cameraOutputUri = null
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