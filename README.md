# What is this?
This is a fan control service for laptops that aims to reduce distracting fan noise as well as smoothing the transitions between speeds.
# How it Works
This program runs as a systemd service in the background to monitor temperatures and adjust fan speeds accordingly. It uses [the acpi-ec driver by MusiKid](https://github.com/MusiKid/acpi_ec) to read from and write to the embedded controller (EC) in order to set fan speeds.
# Why not use NBFC?
NBFC uses Mono Runtime on Linux which is hard to compile and is bloated, taking up excessive amounts of storage. It causes lots of CPU wakeups in the background which can reduce battery life as well as having really basic fan curves and no support for smoothing the transitions between speeds. This program aims to reduce fan noise and smooth the transitions between fan speeds as well as having minimal background CPU usage.
# How to use
```
Usage of fanctl:
  -debug
        Debug output
  -ec-addr int
        Address for fan speed control register (default 25)
  -ec-max float
        Maximum value to write to speed control register (default 48)
  -ec-min float
        Minimum value to write to speed control register
  -ec-path string
        Path to embedded controller interface (default "/dev/ec")
  -manual-addr int
        Address for manual control enable register (default 21)
  -read-addr int
        Address for current speed register (default 17)
  -read-max int
        Maximum value that can be read from current speed register (default 54)
  -read-min int
        Minimum value that can be read from current speed register (default 14)
  -thermal-zone string
        Path to CPU temperature reading (default "/sys/class/hwmon/hwmon5/temp2_input")
```
You can change the options in fanctl.sh.
The defaults for this program are selected for my specific laptop model (HP 15-db1003ca). You may have to change the name and/or number of the CPU thermal zone sensor. If you have a laptop with a similar model or with the same model of EC, the defaults may work.
If it's a different model or vendor, you will need to find the right speed control register address as well as the minimum and maximum speed values and the manual control register address. You can install NBFC and find a working config for your laptop model and look at its contents to find these values. It's also possible that the acpi-ec driver can't access the EC or that none of the registers control the fan speed so this script won't work for you.
# Installation
First, you will need to install and load [the acpi-ec driver by MusiKid](https://github.com/MusiKid/acpi_ec) and make sure that /dev/ec exists.
I created a prebuilt binary for x86 systems for ease of use. If you want, you can compile the program yourself (Optional):
```
rm fanctl
gccgo fanctl.go -o fanctl -O2
strip fanctl
upx --ultra-brute fanctl
```
Then, copy the files to their required destinations.
```
sudo cp fanctl fanctl.sh /usr/bin/
sudo cp fanctl.service /etc/systemd/system
```
Edit the configuration:
```
nano /usr/bin/fanctl.sh
```
Test the service:
```
sudo /usr/bin/fanctl.sh
```
If it controls the fan properly, enable and start the systemd unit.
```
sudo systemctl enable --now fanctl
```
