#!/usr/bin/env python
import json
import os
import re
import time
import subprocess
import shutil
import datetime as dt
import numpy as np
from optparse import OptionParser

#TODO: some paths can be taken from environment variables

def execute_cmd(cmd, f):
    print cmd
    subprocess.call(cmd, stdout=f,stderr=subprocess.STDOUT)

class Query:

    def __init__(self, data, path):
        self.data = data
        self.path = path
        self.tmp_path = path + "/tmp"
        self.data_path = path + "/data"
        if not os.path.exists(self.tmp_path):
            os.makedirs(self.tmp_path)
        if not os.path.exists(self.data_path):
            os.makedirs(self.data_path)

    def getPath(self):
        return self.data_path, self.graph_path

    def getName(self):
        return "query"

    def parseOutput(self, fileList):
        for f in fileList:
            total_bytes = 0
            total_time = 0
            result = []
            for filename in f:
                g = open(filename, "r")
                for line in g:
                    m = re.search("Result: (\w+) query - queries per second (\d+)", line)
                    if m != None:
                        result.append(m.group(2))
            if len(result) > 0:
                rate = max(result)
                std = np.std(result)
                print result
            fname = os.path.basename(f).split("__")[0]
            with open(self.data_path + "/" + fname + "_" + m.group(1), "a+") as myfile:
                myfile.write(time.strftime("%Y%m%d") + "," + str(m.group(2)) + "\n")

    def runCmd(self):
        out = []
        items = self.data["items"]
        config_files = self.data["config"]
        binary_path = self.data["binary"]
        source = self.data["source"]
        qcount = self.data["querycount"]
        target_path = self.tmp_path + "/bench"
        count = self.data["count"]

        for f in config_files:
            if os.path.exists(target_path):
                shutil.rmtree(target_path)
            t = []
            for f in range(count):
                output_file = os.path.basename(f) + ".res" + "__" + str(count)
                CMD = [binary_path, "-config", f, "-target", target_path, "-printTime", "5s", "-source", source, "-count", items, "-querycount", qcount]
                execute_cmd(CMD, open(self.tmp_path + "/" + output_file, "w"))
                t.append(self.tmp_path + "/" + output_file)
            out.append(t)
        return self.parseOutput(out)



class Indexing:

    def __init__(self, data, path):
        self.data = data
        self.path = path
        self.tmp_path = path + "/tmp"
        self.data_path = path + "/data"
        self.graph_path = path + "/graph"
        if not os.path.exists(self.tmp_path):
            os.makedirs(self.tmp_path)
        if not os.path.exists(self.data_path):
            os.makedirs(self.data_path)
        if not os.path.exists(self.graph_path):
            os.makedirs(self.graph_path)

    def getPath(self):
        return self.data_path, self.graph_path

    def getName(self):
        return "indexing"

    def parseOutput(self, fileList):
        for f in fileList:
            total_bytes = 0
            total_time = 0
            result = []
            for filename in f:
                g = open(f, "r")
                for line in g:
                    m = re.search("Result: (\d+) bytes in (\d+) seconds", line)
                    if m != None:
                        total_bytes += int(m.group(1))
                        total_time += int(m.group(2))
                rate = 0
                if total_time > 0:
                    rate = total_bytes/(total_time * 1000)
                    result.append(rate)
            if len(result) > 0:
                rate = max(result)
                std = np.std(result)
                print result
            fname = os.path.basename(f).split("__")[0]
            with open(self.data_path + "/" + fname, "a+") as myfile:
                myfile.write(time.strftime("%Y%m%d") + "," + str(rate) + "\n")

    def runCmd(self):
        out = []
        items = self.data["items"]
        config_files = self.data["config"]
        binary_path = self.data["binary"]
        source = self.data["source"]
        target_path = self.tmp_path + "/bench"
        count = self.data["count"]

        for f in config_files:
            if os.path.exists(target_path):
                shutil.rmtree(target_path)
            t = []
            for f in range(count):
                output_file = os.path.basename(f) + ".res" + "__" + str(count)
                CMD = [binary_path, "-config", f, "-target", target_path, "-printTime", "5s", "-source", source, "-count", items]
                execute_cmd(CMD, open(self.tmp_path + "/" + output_file, "w"))
                t.append(self.tmp_path + "/" + output_file)
            out.append(t)
        return self.parseOutput(out)

class Conf:

    def __init__(self, conf_file, data_path):
        with open(conf_file) as config_file:
            self.config = json.load(config_file)
        self.index_path = data_path + "/" + "index"
        self.query_path = data_path + "/" + "query"

    def run(self):
        self.runIndexing()
        self.runQuery()

    #def saveResult(self, path, result):
    #    pass

    def generateGraph(self, p, name):
        """
        path, gpath = p
        for f in os.listdir(path):
            x = []
            y = []
            for line in open(path + "/" + f, "r"):
                data = line.split()
                x.append(dt.datetime.strptime(data[0],'%d/%m/%Y').date())
                y.append(data[1])
            fig, ax = plt.subplots()
            ax.plot_date(x, y, '-o')
            ax.xaxis.set_major_formatter(mdates.DateFormatter('%m/%d/%Y'))
            ax.xaxis.set_major_locator(mdates.DayLocator())
            ax.autoscale_view()
            ax.grid(True)
            fig.autofmt_xdate()
            ax.set_xlabel('Date')
            ax.set_ylabel('Throughput(MB/sec)')
            plt.savefig(gpath + "/" + f + '.png')
            #plt.show()
            #pylab.savefig(gpath + "/"+ f + ".png")
        """
        pass

    def runIndexing(self):
        if "indexing" in self.config:
            index = Indexing(self.config["indexing"], self.index_path)
            # generate the graphs and return an dict with each type
            index.runCmd()

    def runQuery(self):
        if "query" in self.config:
            query = Query(self.config["query"], self.query_path)
            # generate the graphs and return an dict with each type
            query.runCmd()

if __name__ == "__main__":
    parser = OptionParser()
    parser.add_option("-f", "--file", dest="filename", help="config filename")
    parser.add_option("-d", "--data", dest="data", help="directory to store meta data")
    (options, args) = parser.parse_args()
    if not options.filename:
        parser.error('Filename not given')
    if not options.data:
        parser.error('Data directory path is not given')
    c = Conf(options.filename, options.data)
    c.run()
