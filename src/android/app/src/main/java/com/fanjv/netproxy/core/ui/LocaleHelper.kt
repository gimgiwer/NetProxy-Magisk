package com.fanjv.netproxy.core.ui

import android.content.Context
import android.content.res.Configuration
import android.os.Build
import android.os.LocaleList
import androidx.core.content.edit
import java.util.Locale

/** 应用语言偏好管理与 Context 本地化包装工具。 */
object LocaleHelper {
    const val PREF_KEY_LANGUAGE = "app_language"
    const val LANGUAGE_SYSTEM = "system"
    const val LANGUAGE_ZH_CN = "zh-CN"
    const val LANGUAGE_EN = "en"
    const val LANGUAGE_RU = "ru"

    fun getPersistedLanguage(context: Context): String {
        val prefs = context.getSharedPreferences("settings", Context.MODE_PRIVATE)
        return prefs.getString(PREF_KEY_LANGUAGE, LANGUAGE_SYSTEM) ?: LANGUAGE_SYSTEM
    }

    fun persistLanguage(context: Context, language: String) {
        val prefs = context.getSharedPreferences("settings", Context.MODE_PRIVATE)
        prefs.edit { putString(PREF_KEY_LANGUAGE, language) }
    }

    fun createLocale(language: String): Locale {
        return when (language) {
            LANGUAGE_ZH_CN -> Locale.SIMPLIFIED_CHINESE
            LANGUAGE_EN -> Locale.ENGLISH
            LANGUAGE_RU -> Locale("ru")
            else -> Locale.getDefault()
        }
    }

    fun wrapContext(context: Context, language: String = getPersistedLanguage(context)): Context {
        if (language == LANGUAGE_SYSTEM) {
            return context
        }
        val locale = createLocale(language)
        Locale.setDefault(locale)
        val config = Configuration(context.resources.configuration)
        config.setLocale(locale)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.N) {
            config.setLocales(LocaleList(locale))
        }
        return context.createConfigurationContext(config)
    }

    fun applyLocale(context: Context, language: String = getPersistedLanguage(context)) {
        if (language == LANGUAGE_SYSTEM) return
        val locale = createLocale(language)
        Locale.setDefault(locale)
        val resources = context.resources
        val config = Configuration(resources.configuration)
        config.setLocale(locale)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.N) {
            config.setLocales(LocaleList(locale))
        }
        @Suppress("DEPRECATION")
        resources.updateConfiguration(config, resources.displayMetrics)
    }
}
