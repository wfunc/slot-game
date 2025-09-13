package com.example.gameacmlib;

import android.content.Context;
import android.os.Build;
import android.os.Handler;
import android.os.Looper;
import android.os.Message;
import android.util.Log;

import androidx.annotation.RequiresApi;

import org.eclipse.paho.android.service.MqttAndroidClient;
import org.eclipse.paho.client.mqttv3.IMqttActionListener;
import org.eclipse.paho.client.mqttv3.IMqttDeliveryToken;
import org.eclipse.paho.client.mqttv3.IMqttToken;
import org.eclipse.paho.client.mqttv3.MqttCallback;
import org.eclipse.paho.client.mqttv3.MqttConnectOptions;
import org.eclipse.paho.client.mqttv3.MqttException;
import org.eclipse.paho.client.mqttv3.MqttMessage;
import org.json.JSONException;
import org.json.JSONObject;

import java.io.BufferedReader;
import java.io.File;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

import me.f1reking.serialportlib.SerialPortHelper;
import me.f1reking.serialportlib.entity.DATAB;
import me.f1reking.serialportlib.entity.FLOWCON;
import me.f1reking.serialportlib.entity.PARITY;
import me.f1reking.serialportlib.entity.STOPB;
import me.f1reking.serialportlib.listener.IOpenSerialPortListener;
import me.f1reking.serialportlib.listener.ISerialPortDataListener;
import me.f1reking.serialportlib.listener.Status;
import okhttp3.Call;
import okhttp3.Callback;
import okhttp3.OkHttpClient;
import okhttp3.Request;
import okhttp3.Response;
import okhttp3.ResponseBody;

public class Acmlib {
    private static final String TAG = "Acmlib";
    boolean isdebug = true;

    Sercallback mcallback;
    Handler hander1;
    Handler rbhander;
    int algorepeat = 0;
    int funa = 0;
    int upVP = 0;
    int upVP1 = 0;
    int testcnt = 0;
    int upcnt = 0;
    int vercnt = 0;
    int stacnt = 0;
    String acmversion = "1.2.1";
    String upurl = "http://gzste.top/ACM/";
    String upversion = "1.0.0";
    String VPupurl = "http://gzste.top/VP/";
    String VPtestupurl = "http://gzste.top/VPTest/";
    String VPupurl2 = "http://gzste.top/VP2/";
    String VPurl = "";
    String mqtturl = "http://gzste.top/mqtt/";
    String upVPver = "VP1.0";
    String VPIver = "VP1.0";
    String VPCver = "VP1.0";
    String uid = "";
    int upacm = 0;
    int algoident = 1000;
    String algostr = "";
    int algocnt = 0;
    int algotimecnt = 0;
    int stm32ident = 1000;
    int laststm32ident = 999999999;
    int mqttrenum = 0;
    Context mcontext;
    int upVPcnt = 0;
    byte[] VPdata = new byte[1024 * 256];
    int VPdatalen = 0;
    int msdata = 0;

    public void setRbhander(Handler rbhander) {
        this.rbhander = rbhander;
    }

    public void setdebug(boolean s) {
        isdebug = s;
    }

    class EchoServer2 implements Runnable {
        @Override
        public void run() {


            try {
                //Log.d(TAG, "run: ok");
                switch (funa) {
                    case 0:
                        //mUdpserver.init(hander1, 6002);
                        funa = 1;
                        //intdatamsg();
                        // mUdpClient=new UdpClient(hander1,2);
                        opencomACM();
                        opencom2();
                        break;
                    case 1:
                        if (testcnt > 0) {
                            testcnt--;
                        } else {
                            testcnt = 200;
                            Log.d(TAG, "Service run:" + acmversion + "  algorepeat=" + algorepeat);

                        }
                        break;
                }
                if (update > 0) {
                    switch (update) {
                        case 1:
                            //sta -s gala508 -p RX3189609
                            sendstr("sta -s " + wifiname + " -p " + wifipass + "\r\n");
                            sandupstate(0);
                            update = 2;
                            stacnt = 0;
                            vercnt = 0;
                            break;
                        case 2:
                            if (upcnt > 0) {
                                upcnt--;

                            } else {
                                sendstr("sta\r\n");
                                upcnt = 200;
                                stacnt++;
                                if (stacnt > 10) {
                                    sandupstate(1);
                                    update = 0;
                                    stacnt = 0;
                                }
                            }
                            break;
                        case 3:
                            //ver -u private
                            sendstr("ver -u " + path + "\r\n");
                            sandupstate(2);
                            update = 4;
                            vercnt = 4;
                            break;
                        case 4:
                            if (upcnt > 0) {
                                upcnt--;

                            } else {
                                if (vercnt < 1) {
                                    update = 5;
                                    opencomACM();
                                } else {
                                    sendstr("ver\r\n");
                                    upcnt = 200;
                                    vercnt--;

                                }
                            }
                            break;
                        case 5:
                        case 7:
                            if (upcnt > 0) {
                                upcnt--;

                            } else {

                                sendstr("ver \r\n");
                                upcnt = 200;


                            }
                            break;
                        case 6:
                            if (stacnt > 200) {
                                opencomACM();
                                update = 7;
                            } else {
                                stacnt++;
                            }
                            break;
                    }
                }
                if(mqttindex>0)
                {
                    String messageToSend = mqttbuf[0]; // 保存当前消息
                    
                    // 生成消息ID用于去重
                    try {
                        JSONObject msgObj = new JSONObject(messageToSend);
                        if (msgObj.has("data")) {
                            JSONObject dataObj = msgObj.getJSONObject("data");
                            // 使用消息内容生成唯一ID
                            lastMqttMessageId = dataObj.toString() + "_" + System.currentTimeMillis();
                        }
                    } catch (JSONException e) {
                        lastMqttMessageId = messageToSend.hashCode() + "_" + System.currentTimeMillis();
                    }
                    sendstr2(messageToSend);
                    
                    // 立即移除队列头部消息，不等待响应
                    mqttindex--;
                    System.arraycopy(mqttbuf, 1, mqttbuf, 0, mqttbuf.length - 1);
                    if(isdebug)
                        Log.d(TAG, "MQTT发送消息，队列剩余: " + mqttindex);
                }

            } catch (Throwable e) {

            }
        }
    }
    // MQTT相关变量已简化，不再需要超时和重试机制
    int mqttcnt = 0;
    String filename = "rotateCtrl.udp";

    class EchoServer3 implements Runnable {
        @Override
        public void run() {


            try {


                if (upVP > 0) {
                    switch (upVP) {
                        case 10: {
                            Log.d(TAG, "get VPconfig:" + VPurl + "VPconfig.json");
                            byte[] mdata = downloadFile(VPurl + "VPconfig.json");
                            String md = new String(mdata);
                            Log.d(TAG, "VPconfig:" + md);
                            JSONObject js = new JSONObject(md);
                            String defaultver = "VP1.1";
                            if (js.has("filename")) {
                                filename = js.getString("filename");

                            }
                            if (js.has("default")) {
                                defaultver = js.getString("default");
                                if (VPIver.equals("")) {
                                    upVPver = defaultver;
                                    upVP = 1;
                                    JSONObject rjs = new JSONObject();
                                    try {
                                        rjs.put("MsgType", "M4");
                                        rjs.put("action", "wait");
                                        sendstr2(rjs.toString() + "\r\n");
                                    } catch (JSONException e) {
                                        e.printStackTrace();
                                    }

                                } else {
                                    if (VPCver.equals("")) {
                                        if (VPIver.equals(defaultver)) {
                                            upVP = 0;
                                            try {
                                                JSONObject rjs = new JSONObject();
                                                rjs.put("MsgType", "M4");
                                                rjs.put("action", "exit");
                                                sendstr2(rjs.toString() + "\r\n");
                                            } catch (JSONException e) {
                                                e.printStackTrace();
                                            }
                                        } else {
                                            upVPver = defaultver;
                                            upVP = 1;
                                            try {
                                                JSONObject rjs = new JSONObject();
                                                rjs.put("MsgType", "M4");
                                                rjs.put("action", "wait");
                                                sendstr2(rjs.toString() + "\r\n");
                                            } catch (JSONException e) {
                                                e.printStackTrace();
                                            }
                                        }
                                    } else {
                                        upVPver = VPCver;
                                        upVP = 1;
                                        try {
                                            JSONObject rjs = new JSONObject();
                                            rjs.put("MsgType", "M4");
                                            rjs.put("action", "wait");
                                            sendstr2(rjs.toString() + "\r\n");
                                        } catch (JSONException e) {
                                            e.printStackTrace();
                                        }
                                    }

                                }
                            } else {
                                upVP = 0;
                                try {
                                    JSONObject rjs = new JSONObject();
                                    rjs.put("MsgType", "M4");
                                    rjs.put("action", "exit");
                                    sendstr2(rjs.toString() + "\r\n");
                                } catch (JSONException e) {
                                    e.printStackTrace();
                                }
                            }


                        }

                        break;
                        case 1:
                            upVP = 2;
                            upVP1 = 0;
                            checkVPUpdate();

                            break;
                        case 2: {
                            if (upVPcnt > 0) {
                                upVPcnt--;
                            } else {
                                upVPcnt = 3;
                                try {
                                    JSONObject rjs = new JSONObject();
                                    rjs.put("MsgType", "M4");
                                    rjs.put("action", "wait");
                                    sendstr2(rjs.toString() + "\r\n");
                                } catch (JSONException e) {
                                    e.printStackTrace();
                                }
                            }


                        }
                        break;
                        case 3: {
                            try {
                                JSONObject rjs = new JSONObject();
                                rjs.put("MsgType", "M4");
                                rjs.put("action", "start");
                                sendstr2(rjs.toString() + "\r\n");
                            } catch (JSONException e) {
                                e.printStackTrace();
                            }
                            upVP = 4;


                        }
                        break;
                        case 5: {

                            if (upVP1 == 0) {
                                upVP1 = 1;
                                // 边界检查
                                if (msdata >= 0 && msdata + 128 <= VPdatalen && msdata + 128 <= VPdata.length) {
                                    byte[] str = new byte[128];
                                    System.arraycopy(VPdata, msdata, str, 0, 128);
                                    msdata = msdata + 128;
                                    if (mSerialPortHelper2 != null) {
                                        Log.d("test", "sand to stm32: " + 128);
                                        if (mSerialPortHelper2.isOpen())
                                            mSerialPortHelper2.sendBytes(str);
                                    }
                                } else {
                                    Log.e(TAG, "VP数据边界错误，跳过发送");
                                    upVP = 6;
                                    upVP1 = 0;
                                }
                            } else if (upVP1 == 2) {
                                if (msdata >= 0 && (msdata + 2048) <= VPdatalen && (msdata + 2048) <= VPdata.length) {
                                    byte[] str = new byte[2048];
                                    System.arraycopy(VPdata, msdata, str, 0, 2048);
                                    msdata = msdata + 2048;
                                    if (mSerialPortHelper2 != null) {
                                        Log.d("test", "sand to stm32: " + 2048);
                                        if (mSerialPortHelper2.isOpen())
                                            mSerialPortHelper2.sendBytes(str);
                                    }
                                    upVP1 = 1;
                                } else if (msdata >= 0 && msdata < VPdatalen && msdata < VPdata.length) {
                                    int remainingData = Math.min(VPdatalen - msdata, VPdata.length - msdata);
                                    if (remainingData > 0) {
                                        byte[] str = new byte[remainingData];
                                        System.arraycopy(VPdata, msdata, str, 0, remainingData);
                                        if (mSerialPortHelper2 != null) {
                                            if (isdebug)
                                                Log.d("test", "sand to stm32: " + remainingData);
                                            if (mSerialPortHelper2.isOpen())
                                                mSerialPortHelper2.sendBytes(str);
                                        }
                                        msdata = VPdatalen;
                                    }
                                    upVP = 6;
                                    upVP1 = 0;
                                } else {
                                    upVP = 6;
                                    upVP1 = 0;
                                }
                            }
                        }
                        break;

                    }
                }
                // if(ishasdevid==1)
                //{
                //TOPIC1="mqtt/"+devid+"/sub/02";
                //TOPIC2="mqtt/"+devid+"/pub/02ack";
                //  init();


                // ishasdevid=2;
                // }
                if (ishasdevid == 0 && upVP == 0) {
                    mqttcnt++;
                    if (mqttcnt > 9) {
                        try {
                            JSONObject rjs = new JSONObject();
                            rjs.put("MsgType", "M6");
                            rjs.put("toptype", 0);
                            rjs.put("data", "");
                            sendstr2(rjs.toString() + "\r\n");
                        } catch (JSONException e) {
                            e.printStackTrace();
                        }
                        mqttcnt = 0;
                    }
                }
                if (doConnect && mqttrenum < 20 && shouldReconnect && !isConnecting) {
                    if (mqtthartcnt < 1) {
                        try {
                            boolean needReconnect = false;
                            
                            // 检查客户端状态
                            if (mqttAndroidClient != null) {
                                try {
                                    if (!mqttAndroidClient.isConnected()) {
                                        needReconnect = true;
                                    }
                                } catch (Exception e) {
                                    // 客户端可能已经关闭
                                    Log.w(TAG, "检查MQTT连接状态时出错", e);
                                    needReconnect = true;
                                }
                            } else {
                                // 客户端为空，需要重连
                                needReconnect = true;
                            }
                            
                            if (needReconnect && !isConnecting) {
                                mqttrenum++;
                                Log.i(TAG, "MQTT需要重连，当前重试次数: " + mqttrenum + "/" + 20 + 
                                      ", doConnect=" + doConnect + ", shouldReconnect=" + shouldReconnect);
                                
                                // 延迟执行重连，避免过于频繁
                                new Handler(Looper.getMainLooper()).postDelayed(new Runnable() {
                                    @Override
                                    public void run() {
                                        if (shouldReconnect) {
                                            Log.i(TAG, "开始执行MQTT重连...");
                                            doClientConnection();
                                        } else {
                                            Log.i(TAG, "重连已取消(shouldReconnect=false)");
                                        }
                                    }
                                }, 3000); // 延迟3秒重连
                                mqtthartcnt = 50; // 下次检查间隔30秒
                            } else if (!needReconnect) {
                                // 已连接，重置计数器
                                mqtthartcnt = 50;
                                mqttrenum = 0;
                            }
                        } catch (Exception e) {
                            Log.e(TAG, "MQTT重连检查时发生异常", e);
                            mqtthartcnt = 50;
                        }
                    } else {
                        mqtthartcnt--;
                        // 每10秒输出一次状态
                        if (mqtthartcnt % 100 == 0 && mqtthartcnt > 0) {
                            int seconds = mqtthartcnt / 10;
                            if (seconds > 60) {
                                Log.d(TAG, "MQTT重连倒计时: " + (seconds/60) + "分" + (seconds%60) + "秒");
                            } else {
                                Log.d(TAG, "MQTT重连倒计时: " + seconds + "秒");
                            }
                        }
                    }
                }


            } catch (Throwable e) {

            }
        }
    }

    private void checkVPUpdate() {
        OkHttpClient client = new OkHttpClient();

        Request request = new Request.Builder()
                .url(VPurl + upVPver + "/" + filename)
                .build();
        if (isdebug)
            Log.d(TAG, "开始连接：" + VPurl + upVPver + "/" + filename);
        client.newCall(request).enqueue(new Callback() {
            @Override
            public void onFailure(Call call, IOException e) {
                // 处理失败情况
                if (isdebug)
                    Log.d(TAG, "连接失败");
                try {
                    JSONObject rjs = new JSONObject();
                    rjs.put("MsgType", "M4");
                    rjs.put("action", "fail");
                    sendstr2(rjs.toString() + "\r\n");
                } catch (JSONException ex) {
                    ex.printStackTrace();
                }


                upVP = 0;

            }

            @Override
            public void onResponse(Call call, Response response) throws IOException {
                if (response.isSuccessful()) {
                    //if(isdebug)
                    Log.d(TAG, "开始下载:");
                    byte[] mdata = downloadFile(VPurl + upVPver + "/" + filename);
                    if (mdata != null && mdata.length <= VPdata.length) {
                        System.arraycopy(mdata, 0, VPdata, 0, mdata.length);
                        VPdatalen = mdata.length;
                        msdata = 0;
                        upVP = 3;
                        if (isdebug)
                            Log.d(TAG, "下载完成，大小: " + mdata.length);
                    } else {
                        Log.e(TAG, "下载数据过大或为空，大小: " + (mdata != null ? mdata.length : 0) +
                                " 缓冲区大小: " + VPdata.length);
                        upVP = 0;
                        try {
                            JSONObject rjs = new JSONObject();
                            rjs.put("MsgType", "M4");
                            rjs.put("action", "fail");
                            sendstr2(rjs.toString() + "\r\n");
                        } catch (JSONException ex) {
                            ex.printStackTrace();
                        }
                    }

                } else {
                    if (isdebug)
                        Log.d(TAG, "返回失败:" + response.toString());
                    upVP = 0;
                    try {
                        JSONObject rjs = new JSONObject();
                        rjs.put("MsgType", "M4");
                        rjs.put("action", "exit");
                        sendstr2(rjs.toString() + "\r\n");
                    } catch (JSONException ex) {
                        ex.printStackTrace();
                    }
                }

            }
        });
    }

    private static final OkHttpClient client = new OkHttpClient();

    public byte[] downloadFile(String fileUrl) throws IOException {

        Log.d(TAG, "downloadFile: " + fileUrl);
        Request request = new Request.Builder().url(fileUrl).build();
        try (Response response = client.newCall(request).execute()) {
            if (!response.isSuccessful()) {
                throw new IOException("Unexpected code: " + response);
            }
            ResponseBody body = response.body();
            if (body == null) {
                throw new IOException("Response body is null");
            }
            return body.bytes(); // 直接返回字节数组
        }

    }

    private String downloadJsonBlocking(String urlString) throws Exception {
        HttpURLConnection connection = null;
        BufferedReader reader = null;

        try {
            // 创建URL连接
            URL url = new URL(urlString);
            connection = (HttpURLConnection) url.openConnection();
            connection.setRequestMethod("GET");
            connection.setConnectTimeout(10000); // 10秒连接超时
            connection.setReadTimeout(15000);    // 15秒读取超时

            // 建立连接
            connection.connect();

            // 检查HTTP响应码
            int responseCode = connection.getResponseCode();
            if (responseCode != HttpURLConnection.HTTP_OK) {
                throw new Exception("HTTP错误: " + responseCode);
            }

            // 读取响应数据
            InputStream inputStream = connection.getInputStream();
            reader = new BufferedReader(new InputStreamReader(inputStream));
            StringBuilder response = new StringBuilder();
            String line;

            while ((line = reader.readLine()) != null) {
                response.append(line);
            }

            return response.toString();

        } finally {
            // 清理资源
            if (reader != null) {
                try {
                    reader.close();
                } catch (IOException e) {
                    // 忽略关闭异常
                }
            }
            if (connection != null) {
                connection.disconnect();
            }
        }
    }

    public void contest() {
        opencomACM();
    }

    private ScheduledExecutorService executor2;
    private ScheduledExecutorService executor3;

    public void openacmlib(Context acontext, Sercallback callback) {
        Log.d(TAG, "openacmlib: ");
        mcallback = callback;
        mcontext = acontext;
        hander1 = new Handler() {
            @RequiresApi(api = Build.VERSION_CODES.N)
            @Override
            public void handleMessage(Message msg) {
                super.handleMessage(msg);
                switch (msg.what) {
                    case 2002:
                        opencomACM();
                        break;
                    case 2003:
                        opencom2();
                        break;
                }
            }
        };

        // 确保旧的执行器被正确关闭
        if (executor2 != null && !executor2.isShutdown()) {
            executor2.shutdownNow();
        }
        if (executor3 != null && !executor3.isShutdown()) {
            executor3.shutdownNow();
        }

        executor2 = Executors.newScheduledThreadPool(1);
        executor2.scheduleAtFixedRate(
                new EchoServer2(),
                0,
                10,
                TimeUnit.MILLISECONDS);
        executor3 = Executors.newScheduledThreadPool(1);
        executor3.scheduleAtFixedRate(
                new EchoServer3(),
                0,
                100,
                TimeUnit.MILLISECONDS);
        ishasdevid = 0xff;
        final Handler handler4 = new Handler();
        Runnable runnable4 = new Runnable() {
            @Override
            public void run() {
                //ishasdevid=0;
            }
        };
        handler4.postDelayed(runnable4, 20000);
    }

    // 添加资源清理方法
    // 添加手动重连MQTT的方法
    public void reconnectMqtt() {
        Log.i(TAG, "手动触发MQTT重连");
        shouldReconnect = true;
        isConnecting = false;
        consecutiveFailures = 0; // 重置失败计数
        // 手动重连时先尝试直接重连，只有失败时才重新创建
        clientNeedsRecreate = false;
        mqttrenum = 0; // 重置重连计数
        mqtthartcnt = 10; // 1秒后开始重连
    }
    
    public void shutdown() {
        // 停止重连
        shouldReconnect = false;
        isConnecting = false;
        clientNeedsRecreate = false;
        
        if (executor2 != null && !executor2.isShutdown()) {
            executor2.shutdown();
            try {
                if (!executor2.awaitTermination(2, TimeUnit.SECONDS)) {
                    executor2.shutdownNow();
                }
            } catch (InterruptedException e) {
                executor2.shutdownNow();
                Thread.currentThread().interrupt();
            }
        }

        if (executor3 != null && !executor3.isShutdown()) {
            executor3.shutdown();
            try {
                if (!executor3.awaitTermination(2, TimeUnit.SECONDS)) {
                    executor3.shutdownNow();
                }
            } catch (InterruptedException e) {
                executor3.shutdownNow();
                Thread.currentThread().interrupt();
            }
        }

        // 关闭串口连接
        if (mSerialPortHelper != null) {
            try {
                mSerialPortHelper.close();
            } catch (Exception e) {
                Log.e(TAG, "关闭串口1异常", e);
            } finally {
                mSerialPortHelper = null;
            }
        }
        if (mSerialPortHelper2 != null) {
            try {
                mSerialPortHelper2.close();
            } catch (Exception e) {
                Log.e(TAG, "关闭串口2异常", e);
            } finally {
                mSerialPortHelper2 = null;
            }
        }

        // 关闭MQTT连接
        if (mqttAndroidClient != null) {
            try {
                if (mqttAndroidClient.isConnected()) {
                    mqttAndroidClient.disconnect();
                }
                mqttAndroidClient.unregisterResources();
                mqttAndroidClient.close();
            } catch (Exception e) {
                Log.e(TAG, "关闭MQTT连接异常", e);
            } finally {
                mqttAndroidClient = null;
            }
        }
    }

    public void sandmsg(String str) {
        if (isdebug)
            Log.d(TAG, "from app handle: " + str + "algoident=" + algoident);
        try {
            JSONObject js = new JSONObject();
            if (ishasdevid == 0xff && str.contains("cfgData")) {
                ishasdevid = 0;
            }
            js.put("MsgType", "M1");
            JSONObject mjs = tryParseJson(str);
            if (mjs == null) {
                if (isdebug) Log.e(TAG, "无法解析的JSON: " + str);
                return;
            }
            js.put("data", mjs);

            if (mjs.has("function")) {
                if (mjs.getString("function").equals("algo")) {
                    if (mjs.has("idex")) {
                        if (mjs.getInt("idex") == algoident) {
                            algocnt = 0;
                            algotimecnt = 0;
                            Log.d(TAG, "algo reback: " + mjs.toString());
                        }
                    }
                }
            }

            if (mjs.has("cfgData")){
                try {
                    JSONObject cfgData = mjs.getJSONObject("cfgData");
                    if (cfgData.has("hp30")){
                        int hp30 = cfgData.getInt("hp30");
                        setHp30Conf(hp30);
                    }
                } catch (JSONException e) {
                    Log.e(TAG, "解析 cfgData 错误: " + e.getMessage());
                }
            }


            if (upVP == 0) {
                sendstr2(js.toString() + "\r\n");
            }
        } catch (JSONException e) {
            e.printStackTrace();
        }
    }

    private final Object tDataLock = new Object();
    private final Object t2DataLock = new Object();
    byte[] tdata = new byte[1024 * 10];
    int tlen = 0;
    byte[] t2data = new byte[1024 * 10];
    int t2len = 0;
    int isjm = 0;

    SerialPortHelper mSerialPortHelper;
    SerialPortHelper mSerialPortHelper2;

    // 0 正在连接网络
    //1 正在更新
    //2 更新成功
    //3 更新失败
    public void sandupstate(int state) {
        try {
            JSONObject js = new JSONObject();
            js.put("MsgType", "M3");
            js.put("state", state);
            if (upVP == 0) {
                sendstr2(js.toString() + "\r\n");
            }
        } catch (JSONException e) {
            e.printStackTrace();
        }

    }

    public void sendupreback(int type) {
        try {
            JSONObject rjs = new JSONObject();
            rjs.put("MsgType", "M5");
            rjs.put("upstate", type);
            if (upVP == 0) {
                sendstr2(rjs.toString() + "\r\n");
            }
        } catch (JSONException e) {
            e.printStackTrace();
        }
    }

    private JSONObject tryParseJson(String str) {
        if (str == null || str.isEmpty()) return null;
        str = str.replace("/r", "").replace("/n", "");
        String trimmed = str.trim();
        if (!trimmed.startsWith("{") || !trimmed.endsWith("}")) return null;

        try {
            return new JSONObject(trimmed);
        } catch (JSONException e) {
            if (isdebug) Log.e(TAG, "JSON解析失败: " + str, e);
            return null;
        }
    }

    public void chkresult() {
        synchronized (tDataLock) {
            if (tlen < 2) return;
            if (tdata[tlen - 1] != 0x3e || tdata[tlen - 2] != 0x0a) return;

            byte[] ms = Arrays.copyOf(tdata, tlen);
            tlen = 0;
            String md = new String(ms).replace("\r\n", "").replace(">", "").replace("end", "");

            if (isdebug) Log.d(TAG, "acmreback: " + md);

            if (update > 0) {
                handleUpdateResponse(md);
                return;
            }

            try {
                JSONObject mjs = tryParseJson(md);
                if (mjs == null) {
                    return;  // 跳过这段数据
                }
                JSONObject js = new JSONObject();
                js.put("MsgType", "M2");
                js.put("code", mjs.optInt("code"));
                String fun = mjs.optString("function", "");
                js.put("function", fun);

                if ("algo".equals(fun)) {
                    handleAlgoFunction(mjs, js);
                } else {
                    js.put("data", mjs);
                    sendM2(js);
                }
            } catch (JSONException e) {
                Log.e(TAG, "chkresult JSON parsing error: " + md);
                e.printStackTrace();
            }
        }
    }

    private void handleUpdateResponse(String md) {
        switch (update) {
            case 2:
                if (md.contains("mode")) update = 3;
                break;
            case 4:
                if (md.contains("update")) vercnt = 4;
                if (md.contains("aSTE001")) {
                    sandupstate(4);
                    update = 0;
                }
                break;
            case 5:
                if (md.contains("aSTE001")) {
                    if (md.contains(path)) {
                        sendstr("ver -C\r\n");
                        update = 6;
                        stacnt = 0;
                    } else {
                        sandupstate(4);
                        update = 0;
                    }
                }
                break;
            case 7:
                if (md.contains("aSTE001")) {
                    sandupstate(md.contains(path) ? 3 : 4);
                    update = 0;
                }
                break;
        }
    }

    private int hp30Flag = 0;
    private void setHp30Conf(int hp30){
        hp30Flag = hp30;
    }
    private int hp30OneCounter = 0;
    private void handleAlgoFunction(JSONObject mjs, JSONObject js) throws JSONException {
        if (mjs.has("win")) js.put("win", mjs.getDouble("win"));
        // 处理HP30逻辑
        if (hp30Flag == 1) {
            if (mjs.has("hp30") && mjs.getInt("hp30") == 1) {
                hp30OneCounter++;
                if (hp30OneCounter >= 5) {
                    js.put("hp30", 1);
                    hp30OneCounter = 0; // 重置
                } else {
                    js.put("hp30", 0);
                }
            } else {
                js.put("hp30", 0);
            }
        } else {
            js.put("hp30", 0);
            hp30OneCounter = 0; // 如果不启用hp30，重置计数器
        }

        if (mjs.has("one")) js.put("one", mjs.getInt("one"));
        if (mjs.has("free")) js.put("free", mjs.getInt("free"));
        mjs.put("hp30", js.getInt("hp30")); // 将hp30值放回mjs
        mjs.put("stm32ident", stm32ident);
        mcallback.msgback(mjs.toString() + "\r\n");

        if (upVP == 0) {
            js.put("idex", stm32ident);
            lastalgotostm = js.toString() + "\r\n";
            Message message = new Message();
            message.what = 1;
            message.obj = lastalgotostm;
            rbhander.sendMessage(message);
            sendstr2(lastalgotostm);
        }
    }

    private void sendM2(JSONObject js) throws JSONException {
        if (upVP == 0) {
            js.put("idex", stm32ident);
            lastalgotostm = js.toString() + "\r\n";
            sendstr2(lastalgotostm);
        }
    }


    public boolean resendalgo() {
        if (isdebug)
            Log.d(TAG, "resendalgo: " + lastalgotostm);
        if (lastalgotostm.equals(""))
            return false;
        Message message = new Message();
        message.what = 1;
        message.obj = lastalgotostm;
        rbhander.sendMessage(message);
        sendstr2(lastalgotostm);
        return true;
    }


    // MQTT消息去重相关
    private Map<String, Long> processedMessages = new HashMap<>(); // 存储已处理消息的ID和时间戳
    private static final long MESSAGE_CACHE_DURATION = 30000; // 消息缓存30秒
    private String lastMqttMessageId = ""; // 最后一条MQTT消息的ID
    
    String lastalgotostm="";
    public boolean sendstr(String str)
    {
        if(isdebug)
            Log.d("test", "sandbyteacm: "+str+"updtate="+update);
        if (mSerialPortHelper != null) {


            //Log.d("test", "sandbyteacm2: "+str);
            if(mSerialPortHelper.isOpen())
                mSerialPortHelper.sendBytes(str.getBytes());
        }
        return true;
    }
    private boolean opencomACM()
    {
        Log.d(TAG, "opencomACM: ");
        if (mSerialPortHelper != null) {
            try {
                mSerialPortHelper.close();
            } catch (Exception e) {
                Log.e(TAG, "关闭串口ACM异常", e);
            } finally {
                mSerialPortHelper = null;
            }
        }
        // if (mSerialPortHelper == null) {
        mSerialPortHelper = new SerialPortHelper();
        String[] ls=mSerialPortHelper.getAllDeicesPath();
        int i;
        String path="";
        for(i=0;i<ls.length;i++)
        {
            if(ls[i].contains("ACM"))
            {
                path=ls[i];
                break;
            }
        }
        if(path.equals(""))
        {
            return false;
        }
        mSerialPortHelper.setPort(path);
        mSerialPortHelper.setBaudRate(115200);
        mSerialPortHelper.setStopBits(STOPB.getStopBit(STOPB.B2));
        mSerialPortHelper.setDataBits(DATAB.getDataBit(DATAB.CS8));
        mSerialPortHelper.setParity(PARITY.getParity(PARITY.NONE));
        mSerialPortHelper.setFlowCon(FLOWCON.getFlowCon(FLOWCON.NONE));
        mSerialPortHelper.setmHandler(hander1,2002);
        // }

        // Log.i(TAG, "open: " + Arrays.toString(mSerialPortHelper.getAllDeicesPath()));
        mSerialPortHelper.setIOpenSerialPortListener(new IOpenSerialPortListener() {
            @Override
            public void onSuccess(final File device) {

            }

            @Override
            public void onFail(final File device, final Status status) {

            }
        });
        mSerialPortHelper.setISerialPortDataListener(new ISerialPortDataListener() {
            @Override
            public void onDataReceived(byte[] bytes) {
                if(isdebug)
                    Log.d(TAG, "onDataReceived: "+bytesToHex(bytes));

                // 检查输入参数
                if (bytes == null || bytes.length == 0) {
                    return;
                }

                synchronized (tDataLock) {
                    // 防止缓冲区溢出
                    if (tlen < 0) {
                        tlen = 0;
                    }

                    // 检查剩余空间
                    int remainingSpace = tdata.length - tlen;
                    if (bytes.length > remainingSpace) {
                        Log.w(TAG, "缓冲区空间不足，重置缓冲区");
                        tlen = 0;
                        remainingSpace = tdata.length;
                    }

                    // 如果数据仍然太大，只复制能容纳的部分
                    int copyLength = Math.min(bytes.length, remainingSpace);
                    System.arraycopy(bytes, 0, tdata, tlen, copyLength);
                    tlen += copyLength;

                    if (copyLength < bytes.length) {
                        Log.w(TAG, "数据被截断: " + bytes.length + " -> " + copyLength);
                    }
                }

                chkresult();
            }

            @Override
            public void onDataSend(byte[] bytes) {

            }
        });
        mSerialPortHelper.open();
        return true;
    }
    int update=0;
    String wifiname="";
    String wifipass="";
    String path="";
    int ishasdevid=0xff;
    String devid="123568";
    public void chkresult2() {
        synchronized (t2DataLock) {
            int len = 0, r = 0;
            boolean started = false;

            for (int i = 0; i < t2len; i++) {
                if (!started && t2data[i] == '{') {
                    started = true;
                    r = i;
                }
                if (started && i > 0 && t2data[i] == 0x0A && t2data[i - 1] == 0x0D) {
                    byte[] ms = new byte[i - r - 1];
                    System.arraycopy(t2data, r, ms, 0, ms.length);
                    len = i + 1;
                    started = false;

                    String md = new String(ms);
                    if (isdebug) Log.d(TAG, "from stm32 handle: " + md + "  " + upVP + "  " + upVP1);

                    try {
                        JSONObject js = tryParseJson(md);
                        if (js == null) {
                            Log.e(TAG, "无法解析的JSON: " + md);
                            continue;  // 继续处理下一条消息
                        }
                        if (!js.has("MsgType")) continue;
                        String msgtype = js.getString("MsgType");

                        switch (msgtype) {
                            case "M1":
                                handleMsgM1(js);
                                break;
                            case "M2":
                                handleMsgM2(js);
                                break;
                            case "M3":
                                if (update == 0) handleMsgM3(js);
                                break;
                            case "M4":
                                handleMsgM4(js);
                                break;
                            case "M5":
                                handleMsgM5(js);
                                break;
                            case "M6":
                                handleMsgM6(js);
                                break;
                        }
                    } catch (JSONException e) {
                        Log.e(TAG, "chkresult2 JSON error: " + md);
                        e.printStackTrace();
                    } catch (MqttException e) {
                        Log.e(TAG, "chkresult2 MQTT error: " + md);
                        e.printStackTrace();
                    }
                }
            }

            // 移动未处理的数据到缓冲区开头
            if (len > 0 && len < t2data.length) {
                System.arraycopy(t2data, len, t2data, 0, t2data.length - len);
                t2len -= len;
                if (t2len < 0) t2len = 0;
            }
        }
    }

    private void handleMsgM1(JSONObject js) {
        try {
            JSONObject djs = tryParseJson(js.getString("data"));
            if (djs == null) {
                Log.e(TAG, "无法解析的JSON: " + js.getString("data"));
                return;
            }
            djs.put("acmver", acmversion);

            try {
                if (djs.has("info")){
                    JSONObject info = djs.getJSONObject("info");
                    if (info.has("data") && info.get("data") instanceof JSONObject) {
                        JSONObject data = info.getJSONObject("data");
                        if (data.has("hp30")){
                            int hp30 = data.getInt("hp30");
                            setHp30Conf(hp30);
                        }
                    }
                }
            } catch (JSONException e) {
                Log.e(TAG, "handleMsgM1 error1"+js.toString(), e);
            }

            mcallback.msgback(djs.toString());
        } catch (JSONException e) {
            Log.e(TAG, "handleMsgM1 error2", e);
        }
    }

    private void handleMsgM2(JSONObject js) throws JSONException {
        if (js.has("idex")) {
            stm32ident = js.getInt("idex");
            if (laststm32ident != stm32ident) {
                laststm32ident = stm32ident;
                sendstr(js.getString("data") + "\r\n");
                Message message = new Message();
                message.what = 2;
                message.obj = js.toString();
                rbhander.sendMessage(message);
            } else {
                Message message = new Message();
                message.what = 1;
                message.obj = lastalgotostm;
                rbhander.sendMessage(message);
                sendstr2(lastalgotostm);
                algorepeat++;
            }
        } else {
            sendstr(js.getString("data") + "\r\n");
        }
    }

    private void handleMsgM3(JSONObject js) throws JSONException {
        update = 1;
        wifiname = js.optString("wifiname", "");
        wifipass = js.optString("wifipass", "");
        path = js.optString("path", "");
    }

    private void handleMsgM4(JSONObject js) throws JSONException {
        if (js.has("cVer") && upVP != 10) {
            VPIver = js.optString("lVer", "");
            VPCver = js.optString("cVer", "");

            if (js.has("devType")) {
                uid = js.optString("uid", "");
                String type = js.getString("devType");
                VPurl = "http://gzste.top/update/" + type + "/";
            } else {
                VPurl = js.has("uid") ? VPupurl2 : VPupurl;
            }

            upVP = 10;
            upVP1 = 0;
            sendstr2(new JSONObject().put("MsgType", "M4").put("action", "wait").toString() + "\r\n");
        } else if (js.optString("ready").equals("ok")) {
            if (upVP == 4) {
                upVP = 5;
                upVP1 = 0;
                if (isdebug) Log.d(TAG, "开始发送");
            } else if (upVP == 5 && upVP1 == 1) {
                upVP1 = 2;
                if (isdebug) Log.d(TAG, "再次发送");
            }
        } else if (js.has("error")) {
            upVP = 0;
            upVP1 = 0;
        }
    }

    private void handleMsgM5(JSONObject js) throws JSONException {
        if (js.has("upver")) {
            upversion = js.getString("upver");
            if (upversion.isEmpty()) {
                upversion = "1.0.0";
                upacm = 1;
            } else {
                if (upversion.equals(acmversion)) {
                    sendupreback(0);
                } else {
                    upacm = 1;
                }
            }
        }
    }

    private void handleMsgM6(JSONObject js) throws JSONException, MqttException {
        int toptype = js.getInt("toptype");
        switch (toptype) {
            case 0:
                if (!js.optString("devid").isEmpty()) {
                    int id = js.optInt("mqttserver", 0);
                    if (id == 0) {
                        mqttask++;
                        if (mqttask > 4) ishasdevid = 2;
                    } else {
                        if (js.has("url")) HOST = js.getString("url");
                        if (js.has("name")) USERNAME = js.getString("name");
                        if (js.has("pass")) PASSWORD = js.getString("pass");
                        devid = js.getString("devid");
                        ishasdevid = 1;
                        init();
                    }
                }
                break;
            case 1:
                try {
                    if (mqttAndroidClient != null && mqttAndroidClient.isConnected()) {
                        try {
                            mqttAndroidClient.publish(TOPIC1, js.getString("data").getBytes(), 0, false, null, new IMqttActionListener() {
                                @Override public void onSuccess(IMqttToken asyncActionToken) {
                                    // MQTT发送成功
                                    if(isdebug)
                                        Log.d(TAG, "MQTT TOPIC1发送成功");
                                }
                                @Override public void onFailure(IMqttToken asyncActionToken, Throwable exception) {
                                    if (exception != null) {
                                        Log.e(TAG,"MQTT TOPIC1发送失败: " + exception.getMessage());
                                    } else {
                                        Log.e(TAG,"MQTT TOPIC1发送失败: 未知异常");
                                    }
                                }
                            });
                        } catch (MqttException e) {
                            Log.e(TAG, "MQTT TOPIC1发送异常", e);
                        }
                    }
                } catch (Exception e) {
                    Log.e(TAG, "MQTT TOPIC1检查连接状态异常", e);
                }
                break;
            case 2:
                try {
                    if (mqttAndroidClient != null && mqttAndroidClient.isConnected()) {
                        mqttAndroidClient.publish(TOPIC2, js.getString("data").getBytes(StandardCharsets.UTF_8), 2, false, null, new IMqttActionListener() {
                            @Override public void onSuccess(IMqttToken asyncActionToken) {
                                // MQTT发送成功
                                if(isdebug)
                                    Log.d(TAG, "MQTT TOPIC2发送成功");
                            }
                            @Override public void onFailure(IMqttToken asyncActionToken, Throwable exception) {
                                if (exception != null) {
                                    Log.e(TAG,"MQTT TOPIC2发送失败: " + exception.getMessage());
                                } else {
                                    Log.e(TAG,"MQTT TOPIC2发送失败: 未知异常");
                                }
                            }
                        });
                        Log.d(TAG, "TOPIC2: " + js.getString("data"));
                    }
                } catch (Exception e) {
                    Log.e(TAG, "MQTT TOPIC2检查连接状态异常", e);
                }
                break;
        }
    }

    int mqttask=0;
    public boolean sendstr2(String str)
    {

        if (mSerialPortHelper2 != null) {
            if(isdebug)
                Log.d("test", "sand to stm32: "+str.toString());

            if(mSerialPortHelper2.isOpen())
                mSerialPortHelper2.sendBytes(str.getBytes());
        }
        return true;
    }
    private boolean opencom2()
    {
        if (mSerialPortHelper2 != null) {
            try {
                mSerialPortHelper2.close();
            } catch (Exception e) {
                Log.e(TAG, "关闭串口2异常", e);
            } finally {
                mSerialPortHelper2 = null;
            }
        }
        // if (mSerialPortHelper == null) {
        mSerialPortHelper2 = new SerialPortHelper();
        String[] ls=mSerialPortHelper2.getAllDeicesPath();
        int i;
        String path="/dev/ttyS3";

        mSerialPortHelper2.setPort(path);
        mSerialPortHelper2.setBaudRate(115200);
        mSerialPortHelper2.setStopBits(STOPB.getStopBit(STOPB.B2));
        mSerialPortHelper2.setDataBits(DATAB.getDataBit(DATAB.CS8));
        mSerialPortHelper2.setParity(PARITY.getParity(PARITY.NONE));
        mSerialPortHelper2.setFlowCon(FLOWCON.getFlowCon(FLOWCON.NONE));
        mSerialPortHelper2.setmHandler(hander1,2003);
        // }

        // Log.i(TAG, "open: " + Arrays.toString(mSerialPortHelper.getAllDeicesPath()));
        mSerialPortHelper2.setIOpenSerialPortListener(new IOpenSerialPortListener() {
            @Override
            public void onSuccess(final File device) {

            }

            @Override
            public void onFail(final File device, final Status status) {

            }
        });
        mSerialPortHelper2.setISerialPortDataListener(new ISerialPortDataListener() {
            @Override
            public void onDataReceived(byte[] bytes) {
                // Log.d(TAG, "onDataReceived2: "+bytesToHex(bytes));

                // 检查输入参数
                if (bytes == null || bytes.length == 0) {
                    return;
                }

                synchronized (t2DataLock) {
                    // 防止缓冲区溢出
                    if (t2len < 0) {
                        t2len = 0;
                    }

                    // 检查剩余空间
                    int remainingSpace = t2data.length - t2len;
                    if (bytes.length > remainingSpace) {
                        Log.w(TAG, "t2data缓冲区空间不足，重置缓冲区");
                        t2len = 0;
                        remainingSpace = t2data.length;
                    }

                    // 如果数据仍然太大，只复制能容纳的部分
                    int copyLength = Math.min(bytes.length, remainingSpace);
                    System.arraycopy(bytes, 0, t2data, t2len, copyLength);
                    t2len += copyLength;

                    if (copyLength < bytes.length) {
                        Log.w(TAG, "t2data数据被截断: " + bytes.length + " -> " + copyLength);
                    }
                }

                chkresult2();
            }

            @Override
            public void onDataSend(byte[] bytes) {

            }
        });
        mSerialPortHelper2.open();
        return true;
    }
    public static String bytesToHex(byte[] bytes) {
        StringBuilder sb = new StringBuilder();
        for (byte b : bytes) {
            sb.append(String.format("%02X", b));
        }
        return sb.toString();
    }
    private MqttAndroidClient mqttAndroidClient;
    private MqttConnectOptions mMqttConnectOptions;
    private volatile boolean isConnecting = false; // 添加连接状态标志
    private volatile boolean shouldReconnect = true; // 控制是否应该重连
    private volatile boolean clientNeedsRecreate = false; // 客户端需要重新创建
    private volatile int consecutiveFailures = 0; // 连续失败次数
    public         String             HOST           = "tcp://101.132.118.85:1883";//服务器地址（协议+地址+端口号）
    public         String             USERNAME       = "game";//用户名
    public         String             PASSWORD       = "sdoifj239874fh97g34fdg34";//密码
    public   String             TOPIC1  = "mqtt/MTEST-LYAX3CDDOKM4HLW2HFE305A/sub/02";//发布主题
    public   String             TOPIC2 = "mqtt/MTEST-LYAX3CDDOKM4HLW2HFE305A/pub/02ack";//响应主题

    private void createMqttClient() {
        try {
            Log.i(TAG, "开始创建MQTT客户端: HOST=" + HOST + ", devid=" + devid);
            
            // 使用设备ID作为客户端ID
            mqttAndroidClient = new MqttAndroidClient(mcontext, HOST, devid);
            mqttAndroidClient.setCallback(mqttCallback);

            mMqttConnectOptions = new MqttConnectOptions();
            mMqttConnectOptions.setCleanSession(true);  // 清理会话，确保重连时不会有旧的订阅
            mMqttConnectOptions.setMaxInflight(1024);
            mMqttConnectOptions.setKeepAliveInterval(30);
            mMqttConnectOptions.setAutomaticReconnect(false); // 禁用自动重连
            mMqttConnectOptions.setConnectionTimeout(10); // 10秒超时

            if (!USERNAME.isEmpty()) {
                mMqttConnectOptions.setUserName(USERNAME);
            }
            if (!PASSWORD.isEmpty()) {
                mMqttConnectOptions.setPassword(PASSWORD.toCharArray());
            }
            
            Log.i(TAG, "MQTT客户端创建完成");
        } catch (Exception e) {
            Log.e(TAG, "创建MQTT客户端失败", e);
            mqttAndroidClient = null;
        }
    }


    private void init() {

        String serverURI = HOST; //服务器地址（协议+地址+端口号）
        
        // 重置状态
        shouldReconnect = true;
        isConnecting = false;
        clientNeedsRecreate = false;
        mqttrenum = 0;
        
        createMqttClient();



        // last will message
        doConnect = true;
        /*String message = "{\"terminal_uid\":\"" + CLIENTID + "\"}";
        String topic = PUBLISH_TOPIC;
        Integer qos = 2;
        Boolean retained = false;
        if ((!message.equals("")) || (!topic.equals(""))) {
            // 最后的遗嘱
            try {
                mMqttConnectOptions.setWill(topic, message.getBytes(), qos.intValue(), retained.booleanValue());
            } catch (Exception e) {
                Log.i(TAG, "Exception Occured", e);
                doConnect = false;
                iMqttActionListener.onFailure(null, e);
            }
        }*/
        if (doConnect) {
            // 在主线程执行连接操作
            new Handler(Looper.getMainLooper()).post(new Runnable() {
                @Override
                public void run() {
                    doClientConnection();
                }
            });
        }
    }
    int mqtthartcnt=200;
    boolean doConnect = false;
    String[] mqttbuf=new String[256];
    int mqttindex=0;
    private MqttCallback mqttCallback = new MqttCallback() {

        @Override
        public void messageArrived(String topic, MqttMessage message) throws Exception {
            Log.i(TAG, "收到消息： "+topic+"  " + new String(message.getPayload()));
//            String mstr= new String(message.getPayload());
            mqtthartcnt=200;
            mqttrenum=0;
            
            // 清理过期的消息记录
            cleanExpiredMessages();
            
            if(topic.equals(TOPIC1))
            {
                try {
                    if(mqttindex < mqttbuf.length - 1)  // 更安全的边界检查
                    {
                        JSONObject rjs = new JSONObject();
                        rjs.put("MsgType", "M6");
                        rjs.put("toptype", 1);
                        rjs.put("data", new JSONObject(new String(message.getPayload())));
                        mqttbuf[mqttindex]=rjs.toString()+"\r\n";
                        mqttindex++;
                    } else {
                        Log.w(TAG, "MQTT缓冲区已满，丢弃消息");
                    }
                } catch (JSONException e) {
                    Log.e(TAG, "MQTT消息JSON解析异常", e);
                }
            }
            else if(topic.equals(TOPIC2))
            {
                try {
                    String payload = new String(message.getPayload());
                    JSONObject payloadObj = new JSONObject(payload);
                    
                    // 生成消息标识符
                    String messageKey = topic + "_" + payload;
                    
                    // 检查是否是当前正在发送的消息的ACK
                    if (!lastMqttMessageId.isEmpty()) {
                        lastMqttMessageId = "";
                        if(isdebug) Log.d(TAG, "收到消息的ACK响应");
                    }
                    
                    // 检查是否是重复消息
                    if (processedMessages.containsKey(messageKey)) {
                        if(isdebug) Log.d(TAG, "检测到重复的ACK消息，忽略: " + messageKey);
                        return;
                    }
                    
                    // 记录消息
                    processedMessages.put(messageKey, System.currentTimeMillis());
                    
                    // 只处理非重复的消息
                    JSONObject rjs = new JSONObject();
                    rjs.put("MsgType", "M6");
                    rjs.put("toptype", 2);
                    rjs.put("data", payloadObj);
                    sendstr2(rjs.toString()+"\r\n");
                } catch (JSONException e) {
                    e.printStackTrace();
                }
            }
        }

        @Override
        public void deliveryComplete(IMqttDeliveryToken arg0) {

        }

        @Override
        public void connectionLost(Throwable arg0) {
            Log.i(TAG, "MQTT连接断开: " + (arg0 != null ? arg0.getMessage() : "未知原因"));
            
            // 连接断开，但客户端可能还是好的，先尝试直接重连
            shouldReconnect = true;
            isConnecting = false;
            consecutiveFailures = 0; // 重置失败计数，给直接重连一个机会
            
            // 如果是严重的连接丢失，可能需要清理资源
            if (arg0 != null && arg0.getMessage() != null && 
                (arg0.getMessage().contains("32111") || arg0.getMessage().contains("broken"))) {
                Log.w(TAG, "检测到严重连接问题，清理客户端资源");
                if (mqttAndroidClient != null) {
                    try {
                        mqttAndroidClient.unregisterResources();
                    } catch (Exception e) {
                        Log.w(TAG, "清理连接断开的客户端资源时出错", e);
                    }
                }
                clientNeedsRecreate = true;
                mqtthartcnt = 100; // 10秒后重连
            } else {
                // 正常断线，直接重连
                clientNeedsRecreate = false;
                mqtthartcnt = 50; // 5秒后开始重连
            }
            
            Log.i(TAG, "将在" + (mqtthartcnt/10) + "秒后尝试重连MQTT");
        }
    };
    private void doClientConnection() {
        // 检查是否正在连接或不应该重连
        if (isConnecting || !shouldReconnect) {
            Log.i(TAG, "MQTT连接进行中或已停止重连，跳过本次连接");
            return;
        }
        
        isConnecting = true;
        Log.i(TAG, "开始MQTT连接流程，clientNeedsRecreate=" + clientNeedsRecreate);
        
        try {
            // 检查客户端状态，决定是重连还是重新创建
            if (mqttAndroidClient == null) {
                Log.i(TAG, "客户端为空，创建新的MQTT客户端");
                createMqttClient();
                
                if (mqttAndroidClient == null) {
                    Log.e(TAG, "MQTT客户端创建失败");
                    isConnecting = false;
                    mqtthartcnt = 300;
                    return;
                }
                
                // 新创建的客户端需要等待Service准备
                Log.i(TAG, "等待MQTT Service准备...");
                try {
                    Thread.sleep(3000);
                } catch (InterruptedException ignored) {}
                
                clientNeedsRecreate = false;
            } else if (clientNeedsRecreate) {
                Log.i(TAG, "客户端损坏，需要重新创建");
                
                // 强制重新创建
                try {
                    mqttAndroidClient.unregisterResources();
                    mqttAndroidClient.close();
                } catch (Exception e) {
                    Log.w(TAG, "关闭损坏的客户端时出错", e);
                }
                
                mqttAndroidClient = null;
                
                // 等待资源释放
                try {
                    Thread.sleep(2000);
                    Log.i(TAG, "损坏客户端清理完成");
                } catch (InterruptedException ignored) {}
                
                // 创建新客户端
                createMqttClient();
                if (mqttAndroidClient == null) {
                    Log.e(TAG, "重新创建MQTT客户端失败");
                    isConnecting = false;
                    mqtthartcnt = 300;
                    return;
                }
                
                try {
                    Thread.sleep(3000);
                } catch (InterruptedException ignored) {}
                
                clientNeedsRecreate = false;
            } else {
                // 客户端存在且正常，检查连接状态
                try {
                    if (mqttAndroidClient.isConnected()) {
                        Log.i(TAG, "MQTT客户端已连接，跳过重连");
                        isConnecting = false;
                        return;
                    } else {
                        Log.i(TAG, "客户端存在但未连接，直接重连");
                    }
                } catch (Exception e) {
                    Log.w(TAG, "检查客户端连接状态时出错，标记需要重新创建", e);
                    clientNeedsRecreate = true;
                    isConnecting = false;
                    mqtthartcnt = 100; // 1秒后重试
                    return;
                }
            }

            // 检查是否在等待期间shouldReconnect被设置为false
            if (!shouldReconnect) {
                isConnecting = false;
                return;
            }

            // 尝试连接
            try {
                Log.i(TAG, "开始MQTT连接...");
                mqttAndroidClient.connect(mMqttConnectOptions, null, iMqttActionListener);
            } catch (MqttException e) {
                Log.e(TAG, "MQTT连接异常: " + e.getReasonCode() + ", " + e.getMessage(), e);
                
                if (e.getReasonCode() == 32111) {
                    // 客户端已关闭，递增重试间隔
                    consecutiveFailures++;
                    int retryDelay = Math.min(consecutiveFailures * 100, 600); // 递增延迟，最多60秒
                    
                    Log.e(TAG, "MQTT客户端已关闭(32111)，连续失败" + consecutiveFailures + "次，" + (retryDelay/10) + "秒后重试");
                    
                    // 立即清理客户端
                    if (mqttAndroidClient != null) {
                        try {
                            mqttAndroidClient.unregisterResources();
                            mqttAndroidClient.close();
                        } catch (Exception ex) {
                            Log.w(TAG, "清理32111错误的客户端时出错", ex);
                        }
                        mqttAndroidClient = null;
                    }
                    
                    clientNeedsRecreate = true;
                    mqtthartcnt = retryDelay;
                } else if (e.getReasonCode() == 32103) {
                    // 网络连接问题
                    Log.e(TAG, "网络连接问题，30秒后重试");
                    mqtthartcnt = 50;
                } else {
                    mqtthartcnt = 50; // 20秒后重试
                }
                isConnecting = false;
            }
        } catch (Exception e) {
            Log.e(TAG, "MQTT连接时发生未知异常", e);
            clientNeedsRecreate = true;
            isConnecting = false;
            mqtthartcnt = 50;
        }
    }



    //MQTT是否连接成功的监听
    private IMqttActionListener iMqttActionListener = new IMqttActionListener() {

        @Override
        public void onSuccess(IMqttToken arg0) {
            Log.i(TAG, "MQTT连接成功");
            isConnecting = false;
            mqttrenum = 0; // 重置重连计数
            mqtthartcnt = 50; // 重置心跳计数
            consecutiveFailures = 0; // 重置连续失败计数

            // 延迟订阅，确保连接稳定
            new Handler(Looper.getMainLooper()).postDelayed(new Runnable() {
                @Override
                public void run() {
                    try {
                        if (mqttAndroidClient != null && mqttAndroidClient.isConnected()) {
                            TOPIC1="mqtt/"+devid+"/sub/02";
                            TOPIC2="mqtt/"+devid+"/pub/02ack";

                            // 分别订阅，避免一个失败影响另一个
                            try {
                                mqttAndroidClient.subscribe(TOPIC1, 1, null, new IMqttActionListener() {
                                    @Override
                                    public void onSuccess(IMqttToken asyncActionToken) {
                                        Log.i(TAG, "成功订阅 " + TOPIC1);
                                    }
                                    @Override
                                    public void onFailure(IMqttToken asyncActionToken, Throwable exception) {
                                        if (exception != null) {
                                            Log.e(TAG, "订阅 " + TOPIC1 + " 失败" + exception.getMessage());
                                        } else {
                                            Log.e(TAG,"订阅 " + TOPIC1 + " 失败未知异常");
                                        }
                                    }
                                });
                            } catch (Exception e) {
                                Log.e(TAG, "订阅TOPIC1异常", e);
                            }

                            try {
                                mqttAndroidClient.subscribe(TOPIC2, 1, null, new IMqttActionListener() {
                                    @Override
                                    public void onSuccess(IMqttToken asyncActionToken) {
                                        Log.i(TAG, "成功订阅 " + TOPIC2);
                                    }
                                    @Override
                                    public void onFailure(IMqttToken asyncActionToken, Throwable exception) {
                                        if (exception != null) {
                                            Log.e(TAG, "订阅 " + TOPIC2 + " 失败" + exception.getMessage());
                                        } else {
                                            Log.e(TAG,"订阅 " + TOPIC2 + " 失败未知异常");
                                        }
                                    }
                                });
                            } catch (Exception e) {
                                Log.e(TAG, "订阅TOPIC2异常", e);
                            }
                        }
                    } catch (Exception e) {
                        Log.e(TAG, "MQTT订阅过程异常", e);
                    }
                }
            }, 500); // 延迟500ms
        }

        @Override
        public void onFailure(IMqttToken arg0, Throwable arg1) {
            isConnecting = false;
            
            // 检查错误类型
            if (arg1 instanceof MqttException) {
                MqttException mqttException = (MqttException) arg1;
                int reasonCode = mqttException.getReasonCode();
                Log.e(TAG, "MQTT连接失败，错误码: " + reasonCode + ", 消息: " + mqttException.getMessage());
                
                if (reasonCode == 32111) {
                    // 客户端已关闭，递增重试间隔
                    consecutiveFailures++;
                    int retryDelay = Math.min(consecutiveFailures * 100, 600); // 递增延迟，最多60秒
                    
                    Log.e(TAG, "MQTT连接失败(32111)，连续失败" + consecutiveFailures + "次，" + (retryDelay/10) + "秒后重试");
                    
                    // 立即清理客户端，防止重复回调
                    if (mqttAndroidClient != null) {
                        try {
                            mqttAndroidClient.unregisterResources();
                            mqttAndroidClient.close();
                        } catch (Exception e) {
                            Log.w(TAG, "清理失败的MQTT客户端时出错", e);
                        }
                        mqttAndroidClient = null;
                    }
                    
                    clientNeedsRecreate = true;
                    mqtthartcnt = retryDelay;
                } else if (reasonCode == 32103) {
                    // 无法连接到服务器
                    Log.e(TAG, "无法连接到MQTT服务器，可能是网络问题");
                    mqtthartcnt = 50; // 30秒后重试
                } else {
                    // 其他错误
                    Log.e(TAG, "MQTT连接失败，将继续重试");
                    mqtthartcnt = 50; // 20秒后重试
                }
            } else {
                Log.e(TAG, "MQTT连接失败: " + (arg1 != null ? arg1.getMessage() : "未知错误"));
                mqtthartcnt = 50;
            }
        }
    };
    // 清理过期的消息记录
    private void cleanExpiredMessages() {
        long currentTime = System.currentTimeMillis();
        processedMessages.entrySet().removeIf(entry -> 
            currentTime - entry.getValue() > MESSAGE_CACHE_DURATION);
    }
    
    public interface Sercallback{
        public void msgback(String str);
    }
}
