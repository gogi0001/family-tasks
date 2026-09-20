package com.gogi0001.familytasks

import android.app.Activity
import android.content.Intent
import android.os.Bundle
import android.widget.Button
import android.widget.EditText
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity

class SettingsActivity : AppCompatActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_settings)
        supportActionBar?.setDisplayHomeAsUpEnabled(true)

        val prefs = getSharedPreferences("app", MODE_PRIVATE)
        val input = findViewById<EditText>(R.id.server_url)

        val current = prefs.getString("serverUrl", "").orEmpty()
        input.setText(current.ifEmpty { "http://10.0.2.2:8787" })
        // Ставим курсор в конец, чтобы правка была быстрой.
        input.setSelection(input.text.length)

        findViewById<Button>(R.id.btn_save).setOnClickListener {
            val value = input.text.toString().trim().trimEnd('/')
            if (!value.startsWith("http://") && !value.startsWith("https://")) {
                Toast.makeText(
                    this,
                    getString(R.string.error_bad_url, value),
                    Toast.LENGTH_SHORT
                ).show()
                return@setOnClickListener
            }
            prefs.edit().putString("serverUrl", value).apply()

            // Возвращаем новый URL в MainActivity, чтобы перезагрузить сразу.
            val result = Intent().putExtra("serverUrl", value)
            setResult(Activity.RESULT_OK, result)
            finish()
        }

        findViewById<Button>(R.id.btn_cancel).setOnClickListener {
            setResult(Activity.RESULT_CANCELED)
            finish()
        }

        findViewById<Button>(R.id.btn_reset).setOnClickListener {
            prefs.edit().remove("serverUrl").apply()
            input.setText("http://10.0.2.2:8787")
            input.setSelection(input.text.length)
        }
    }

    override fun onSupportNavigateUp(): Boolean {
        setResult(Activity.RESULT_CANCELED)
        finish()
        return true
    }
}