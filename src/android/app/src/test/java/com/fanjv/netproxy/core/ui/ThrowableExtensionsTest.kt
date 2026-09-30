package com.fanjv.netproxy.core.ui

import kotlinx.coroutines.CancellationException
import org.junit.Assert.assertEquals
import org.junit.Test
import java.util.Locale

class ThrowableExtensionsTest {
    @Test(expected = CancellationException::class)
    fun `cancellation is never converted to a user error`() {
        CancellationException("qz2 was cancelled").userMessage()
    }

    @Test
    fun `regular errors keep their readable message and translate appropriately`() {
        val originalLocale = Locale.getDefault()
        try {
            Locale.setDefault(Locale.SIMPLIFIED_CHINESE)
            assertEquals("读取状态失败", IllegalStateException("读取状态失败").userMessage())

            Locale.setDefault(Locale.ENGLISH)
            assertEquals("Failed to read status", IllegalStateException("读取状态失败").userMessage())

            Locale.setDefault(Locale("ru", "RU"))
            assertEquals("Не удалось прочитать статус", IllegalStateException("读取状态失败").userMessage())
        } finally {
            Locale.setDefault(originalLocale)
        }
    }
}
