// test789d4 : USAG Lib basic io Abstract File Android

import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;

import android.content.Context;
import android.net.Uri;
import android.provider.OpenableColumns;
import android.database.Cursor;

public class AFileAndroid implements AFile {
    private final Context context;
    private final Uri uri;
    private String nameCache = null;
    private long sizeCache = -1;

    public AFileAndroid(Context context, Uri uri) {
        this.context = context;
        this.uri = uri;
    }

    @Override
    public InputStream openInput() throws IOException {
        return context.getContentResolver().openInputStream(uri);
    }

    @Override
    public OutputStream openOutput() throws IOException {
        return context.getContentResolver().openOutputStream(uri);
    }

    @Override
    public String getName() {
        if (nameCache != null) return nameCache;
        // Logic from FileUtil.getFileName
        try (Cursor cursor = context.getContentResolver().query(uri, 
                new String[]{OpenableColumns.DISPLAY_NAME}, null, null, null)) {
            if (cursor != null && cursor.moveToFirst()) {
                int idx = cursor.getColumnIndex(OpenableColumns.DISPLAY_NAME);
                if (idx != -1) nameCache = cursor.getString(idx);
            }
        }
        return nameCache;
    }

    @Override
    public long getSize() {
        if (sizeCache != -1) return sizeCache;
        // Logic from FileUtil.GetFileSize
        try (Cursor cursor = context.getContentResolver().query(uri, 
                new String[]{OpenableColumns.SIZE}, null, null, null)) {
            if (cursor != null && cursor.moveToFirst()) {
                int idx = cursor.getColumnIndex(OpenableColumns.SIZE);
                if (idx != -1) sizeCache = cursor.getLong(idx);
            }
        }
        return sizeCache;
    }
    
    @Override
    public boolean isDirectory() {
        // MIME type check or DocumentContract check needed for robust dir checking
        return false; // Simply false for single file wrapper
    }
}

///// 실제 안드로이드 개발하면서 업데이트 필요 /////