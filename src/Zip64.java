// test789d5 : USAG Lib basic io Zip64

import java.io.BufferedInputStream;
import java.io.BufferedOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.util.ArrayList;
import java.util.List;
import java.util.zip.ZipEntry;
import java.util.zip.ZipInputStream;
import java.util.zip.ZipOutputStream;

public class Zip64 {
    // Z64Writer
    private OutputStream originalStream;
    private ZipOutputStream zout;

    //Z64Reader
    private AFile source;

    public Zip64(AFile f, byte[] header, boolean compress, boolean isReading) throws IOException {
        this.originalStream = null;
        this.zout = null;
        this.source = null;
        if (isReading) {
            this.source = f;
        } else {
            this.originalStream = f.openOutput();
            if (header != null && header.length > 0) {
                this.originalStream.write(header);
            }
            this.zout = new ZipOutputStream(new BufferedOutputStream(this.originalStream));
            this.zout.setLevel(compress ? 9 : 0);
        }
    }

    public void close() throws IOException {
        if (this.zout != null) {
            this.zout.close(); // caller will close originalStream
        }
    }

    public void writeFile(String name, AFile src) throws IOException {
        ZipEntry entry = new ZipEntry(name);
        zout.putNextEntry(entry);
            
        try (InputStream in = new BufferedInputStream(source.openInput())) {
            byte[] buffer = new byte[1048576];
            int len;
            while ((len = in.read(buffer)) != -1) {
                zout.write(buffer, 0, len);
            }
        }
        zout.closeEntry();
    }

    public void writeBin(String name, byte[] data) throws IOException {
        ZipEntry entry = new ZipEntry(name);
        zout.putNextEntry(entry);
        zout.write(data);
        zout.closeEntry();
    }

    public interface UnzipCallback { // callback to do with each entry
        AFile onEntryFound(String name);
    }

    public String[] open(AFile destDirFactory, UnzipCallback callback) throws IOException {
        List<String> nameList = new ArrayList<>();
        try (ZipInputStream zin = new ZipInputStream(new BufferedInputStream(source.openInput()))) {
            ZipEntry entry;
            byte[] buffer = new byte[1048576];
                
            while ((entry = zin.getNextEntry()) != null) {
                nameList.add(entry.getName());
                if (entry.isDirectory()) {
                    continue;
                }
                if (callback == null) {
                    continue;
                }

                AFile outFile = callback.onEntryFound(entry.getName());
                try (OutputStream out = new BufferedOutputStream(outFile.openOutput())) {
                    int len;
                    while ((len = zin.read(buffer)) != -1) {
                        out.write(buffer, 0, len);
                    }
                }
                zin.closeEntry();
            }
        }
        return nameList.toArray(new String[0]);
    }
}