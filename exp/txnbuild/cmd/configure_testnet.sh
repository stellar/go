#!/bin/bash

# Use the lumen command line tool to set up three test accounts,
# and one to use for account merge operations to destroy them.
# Initial balances will be
# test0: 19900
# test1: 10000
# test2: 100
# devnull: 10000

test0=GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3
test0sign=SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R

test1=GAS4V4O2B7DW5T7IQRPEEVCRXMDZESKISR7DVIGKZQYYV3OSQ5SH5LVP
test1sign=SBMSVD4KKELKGZXHBUQTIROWUAPQASDX7KEJITARP4VMZ6KLUHOGPTYW

test2=GB7BDSZU2Y27LYNLALKKALB52WS2IZWYBDGY6EQBLEED3TJOCVMZRH7H
test2sign=SBZVMB74Z76QZ3ZOY7UTDFYKMEGKW5XFJEB6PFKBF4UYSSWHG4EDH7PY

devnull=GBAQPADEYSKYMYXTMASBUIS5JI3LMOAWSTM2CHGDBJ3QDDPNCSO3DVAA
devnullsign=SD3ZKHOPXV6V2QPLCNNH7JWGKYWYKDFPFRNQSKSFF3Q5NJFPAB5VSO6D

lumen balance $test0
if [ $? -ne 0 ];
then
    echo Creating account test0=$test0
    lumen friendbot $test0
    lumen account set test0 $test0
    lumen account set test0sign $test0sign
else
    echo Account test0 already exists
fi

lumen balance $test1
if [ $? -ne 0 ];
then
    echo Creating account test1=$test1
    lumen friendbot $test1
    lumen account set test1 $test1
    lumen account set test1sign $test1sign
else
    echo Account test1 already exists
fi

lumen balance $test2
if [ $? -ne 0 ];
then
    echo Creating account test2=$test2
    lumen friendbot $test2
    lumen account set test2 $test2
    lumen account set test2sign $test2sign
    lumen pay 9900 --from test2sign --to test0
else
    echo Account test2 already exists
fi

lumen balance $devnull
if [ $? -ne 0 ];
then
    echo Creating account devnull=$devnull
    lumen friendbot $devnull
    lumen account set devnull $devnull
    lumen account set devnullsign $devnullsign
else
    echo Account devnull already exists
fi
